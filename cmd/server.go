package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/nghiack7/game-ad-service/internal/api"
	"github.com/nghiack7/game-ad-service/internal/config"
	"github.com/nghiack7/game-ad-service/internal/processor"
	"github.com/nghiack7/game-ad-service/internal/service"
	"github.com/nghiack7/game-ad-service/internal/storage"
	"github.com/nghiack7/game-ad-service/internal/storage/cache"
	"github.com/nghiack7/game-ad-service/internal/storage/memory"
	"github.com/nghiack7/game-ad-service/pkg/logger"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the API server",
	Run:   serverRun,
}

func init() {
	rootCmd.AddCommand(serverCmd)
}

// Serve is the function that runs the API server
func serverRun(cmd *cobra.Command, args []string) {

	// Initialize logger
	logInstance := logger.NewLogger(config.AppConfig)

	logInstance.Infof("Starting Game Ad Service API")

	// Generate instance ID
	instanceID := fmt.Sprintf("api-%s", uuid.New().String()[:8])
	logInstance.Infof("Instance ID generated: %s", instanceID)

	// Initialize components
	components, err := initializeComponents(config.AppConfig, instanceID, logInstance)
	if err != nil {
		logInstance.Fatalf(err, "Failed to initialize components")
	}

	// Create API router
	router := api.NewRouter(components.adService, logInstance)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.AppConfig.Server.Port),
		Handler:      router,
		ReadTimeout:  config.AppConfig.Server.ReadTimeout,
		WriteTimeout: config.AppConfig.Server.WriteTimeout,
		IdleTimeout:  config.AppConfig.Server.IdleTimeout,
	}

	// Start queue maintenance in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start queue maintenance loop in background
	go components.queue.StartMaintenanceLoop(ctx)

	// Start a minimal worker pool for processing high-priority items directly in API server
	// This allows for immediate processing of urgent items without waiting for worker nodes
	if config.AppConfig.Processor.Worker.MinWorkers > 0 {
		workerPoolOptions := processor.DefaultWorkerPoolOptions()
		workerPoolOptions.MinWorkers = 1          // Use just one worker in API server
		workerPoolOptions.MaxWorkers = 2          // Allow up to 2 workers if needed
		workerPoolOptions.EnableAutoScale = false // Don't auto-scale in API server

		components.workerPool = processor.NewWorkerPool(
			components.queue,
			components.analyzer,
			workerPoolOptions,
		)

		if err := components.workerPool.Start(ctx); err != nil {
			logInstance.Errorf("Failed to start minimal worker pool: %v", err)
		} else {
			logInstance.Infof("Started minimal worker pool for high-priority processing")
		}
	}

	// Start server in a goroutine
	go func() {
		logInstance.Infof("Starting HTTP server on port %d", config.AppConfig.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logInstance.Fatalf(err, "Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logInstance.Infof("Received signal %v, shutting down server...", sig)

	// Create a deadline for server shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown the server
	if err := server.Shutdown(shutdownCtx); err != nil {
		logInstance.Fatalf(err, "Server shutdown failed: %v", err)
	}

	// Cancel context to stop all workers and queue maintenance
	cancel()

	// Close queue
	if err := components.queue.Close(); err != nil {
		logInstance.Errorf("Error closing queue: %v", err)
	}

	logInstance.Infof("Server shutdown complete")
}

// Components holds all the application components
type Components struct {
	memoryRepo *memory.MemoryRepository
	redisRepo  *cache.RedisRepository
	queue      *processor.DistributedPriorityQueue
	analyzer   *processor.Analyzer
	workerPool *processor.WorkerPool
	adService  service.AdService
}

// Initialize all the components
func initializeComponents(cfg *config.Config, instanceID string, logInstance logger.Logger) (*Components, error) {
	components := &Components{}

	// Create memory repository
	components.memoryRepo = memory.NewMemoryRepository() // TODO currently use memory as repo
	logInstance.Infof("Memory repository initialized")

	// Initialize Redis if configured
	if cfg.Storage.CacheType == "redis" || cfg.Storage.PersistenceType == "redis" {
		redisOptions := cache.RedisOptions{
			Address:    cfg.Storage.Redis.Address,
			Password:   cfg.Storage.Redis.Password,
			DB:         cfg.Storage.Redis.DB,
			DefaultTTL: time.Duration(cfg.Storage.CacheTTL) * time.Second,
		}

		var err error
		components.redisRepo, err = cache.NewRedisRepository(redisOptions)
		if err != nil {
			logInstance.Errorf("Failed to connect to Redis, using in-memory cache only: %v", err)
			components.redisRepo = nil
		} else {
			logInstance.Infof("Redis connected to %s", cfg.Storage.Redis.Address)
		}
	}

	// Create analyzer
	analyzerOptions := processor.AnalyzerOptions{
		MinScore:      1.0,
		MaxScore:      10.0,
		AnalysisDelay: time.Duration(0), // No artificial delay
	}
	components.analyzer = processor.NewAnalyzer(analyzerOptions)
	logInstance.Infof("Analyzer initialized")

	// Determine which repository to use as cache
	var cacheRepo storage.CacheRepository
	if cfg.Storage.CacheType == "redis" && components.redisRepo != nil {
		cacheRepo = components.redisRepo
		logInstance.Infof("Using Redis for cache repository")
	} else {
		cacheRepo = components.memoryRepo
		logInstance.Infof("Using in-memory repository for cache")
	}

	// Create queue
	queueOptions := processor.DefaultQueueOptions()
	queueOptions.MaxCapacity = cfg.Processor.QueueCapacity
	queueOptions.ProcessingTimeout = cfg.Processor.ProcessingTimeout
	queueOptions.MaxAttempts = cfg.Processor.MaxAttempts
	queueOptions.LockTTL = time.Duration(cfg.Processor.LockTTL) * time.Second
	queueOptions.EnableAging = true // Enable priority aging to prevent starvation

	components.queue = processor.NewDistributedPriorityQueue(
		components.memoryRepo,
		cacheRepo,
		instanceID,
		queueOptions,
	)
	logInstance.Infof("Distributed priority queue initialized")

	// Create worker pool for API server
	workerPoolOptions := processor.WorkerPoolOptions{
		WorkerOptions: processor.WorkerOptions{
			MaxConcurrent:  cfg.Processor.Worker.MaxConcurrentJobs,
			ProcessTimeAvg: 2 * time.Second, // Average time to process an ad
		},
		MinWorkers:      cfg.Processor.Worker.MinWorkers,
		MaxWorkers:      cfg.Processor.Worker.MaxWorkers,
		EnableAutoScale: cfg.Processor.Worker.EnableAutoScale,
		TargetQueueSize: cfg.Processor.Worker.TargetQueueSize,
	}

	components.workerPool = processor.NewWorkerPool(
		components.queue,
		components.analyzer,
		workerPoolOptions,
	)
	logInstance.Infof("Worker pool initialized")

	// Create ad service
	components.adService = service.NewAdService(
		components.memoryRepo,
		cacheRepo,
		components.queue,
		components.workerPool,
	)
	logInstance.Infof("Ad service initialized")

	return components, nil
}
