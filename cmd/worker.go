package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nghiack7/game-ad-service/internal/config"
	"github.com/nghiack7/game-ad-service/internal/processor"
	"github.com/nghiack7/game-ad-service/internal/storage/cache"
	"github.com/nghiack7/game-ad-service/internal/storage/memory"
	"github.com/nghiack7/game-ad-service/pkg/logger"
	"github.com/nghiack7/game-ad-service/pkg/utils"
	"github.com/spf13/cobra"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Start the worker",
	Run:   workerRun,
}

func init() {
	rootCmd.AddCommand(workerCmd)
}

func workerRun(cmd *cobra.Command, args []string) {
	logInstance := logger.NewLogger(config.AppConfig)

	logInstance.Infof("Starting Game Ad Service Worker")

	// Generate instance ID
	instanceID := utils.GenerateInstanceID("worker")
	logInstance.Infof("Instance ID generated: %s", instanceID)

	// Initialize repositories
	memoryRepo := memory.NewMemoryRepository()

	// Convert Redis config to Redis options
	redisOptions := cache.RedisOptions{
		Address:    config.AppConfig.Storage.Redis.Address,
		Password:   config.AppConfig.Storage.Redis.Password,
		DB:         config.AppConfig.Storage.Redis.DB,
		DefaultTTL: time.Duration(config.AppConfig.Storage.CacheTTL) * time.Second,
	}

	redisRepo, err := cache.NewRedisRepository(redisOptions)
	if err != nil {
		logInstance.Fatalf(err, "Failed to initialize Redis repository")
	}

	// Initialize queue
	queueOptions := processor.DefaultQueueOptions()
	queueOptions.MaxCapacity = config.AppConfig.Processor.QueueCapacity
	queueOptions.ProcessingTimeout = config.AppConfig.Processor.ProcessingTimeout
	queueOptions.MaxAttempts = config.AppConfig.Processor.MaxAttempts
	queueOptions.LockTTL = time.Duration(config.AppConfig.Processor.LockTTL) * time.Second

	queue := processor.NewDistributedPriorityQueue(
		memoryRepo,
		redisRepo,
		instanceID,
		queueOptions,
	)

	// Initialize analyzer with default options
	analyzerOptions := processor.DefaultAnalyzerOptions()
	analyzer := processor.NewAnalyzer(analyzerOptions)

	// Initialize worker pool options
	workerPoolOptions := processor.DefaultWorkerPoolOptions()
	workerPoolOptions.MinWorkers = config.AppConfig.Processor.Worker.MinWorkers
	workerPoolOptions.MaxWorkers = config.AppConfig.Processor.Worker.MaxWorkers
	workerPoolOptions.EnableAutoScale = config.AppConfig.Processor.Worker.EnableAutoScale
	workerPoolOptions.TargetQueueSize = config.AppConfig.Processor.Worker.TargetQueueSize

	// Initialize worker pool
	workerPool := processor.NewWorkerPool(
		queue,
		analyzer,
		workerPoolOptions,
	)

	// Create context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start maintenance loop for queue
	go queue.StartMaintenanceLoop(ctx)

	// Start worker pool
	if err := workerPool.Start(ctx); err != nil {
		logInstance.Errorf("Failed to start worker pool: %v", err)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	sig := <-sigChan
	logInstance.Infof("Received signal %v, shutting down...", sig)

	// Cancel context to stop all workers
	cancel()

	// Close queue
	if err := queue.Close(); err != nil {
		logInstance.Errorf("Error closing queue: %v", err)
	}

	logInstance.Infof("Worker shutdown complete")
}
