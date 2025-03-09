package processor

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/nghiack7/game-ad-service/internal/domain/models"
)

// Worker processes ads from the queue
type Worker struct {
	id            string
	queue         *DistributedPriorityQueue
	analyzer      *Analyzer
	maxConcurrent int
	processTime   time.Duration

	// Status tracking
	isRunning  atomic.Bool
	activeJobs atomic.Int32
	processed  atomic.Int64
	errors     atomic.Int64

	// For shutdown signaling
	shutdownCh chan struct{}
	wg         sync.WaitGroup
}

// WorkerOptions configures worker behavior
type WorkerOptions struct {
	// MaxConcurrent is the maximum number of concurrent jobs per worker
	MaxConcurrent int

	// ProcessTimeAvg is the average time to process an ad (randomized +/- 30%)
	ProcessTimeAvg time.Duration
}

// DefaultWorkerOptions returns sensible defaults for worker options
func DefaultWorkerOptions() WorkerOptions {
	return WorkerOptions{
		MaxConcurrent:  runtime.NumCPU(),
		ProcessTimeAvg: 2 * time.Second,
	}
}

// NewWorker creates a new worker
func NewWorker(
	queue *DistributedPriorityQueue,
	analyzer *Analyzer,
	options WorkerOptions,
) *Worker {
	// Generate a unique worker ID
	workerID := fmt.Sprintf("worker-%s", uuid.New().String())

	return &Worker{
		id:            workerID,
		queue:         queue,
		analyzer:      analyzer,
		maxConcurrent: options.MaxConcurrent,
		processTime:   options.ProcessTimeAvg,
		shutdownCh:    make(chan struct{}),
	}
}

// Start begins processing ads from the queue
func (w *Worker) Start(ctx context.Context) error {
	if w.isRunning.Swap(true) {
		return fmt.Errorf("worker %s is already running", w.id)
	}

	// Launch worker goroutines
	for i := 0; i < w.maxConcurrent; i++ {
		w.wg.Add(1)
		go w.processLoop(ctx, i)
	}

	return nil
}

// Stop gracefully shuts down the worker
// If waitForCompletion is true, it waits for all current jobs to complete
func (w *Worker) Stop(waitForCompletion bool) error {
	if !w.isRunning.Swap(false) {
		return nil // Already stopped
	}

	close(w.shutdownCh)

	if waitForCompletion {
		w.wg.Wait()
	}

	return nil
}

// Stats returns current worker statistics
func (w *Worker) Stats() map[string]interface{} {
	return map[string]interface{}{
		"id":            w.id,
		"running":       w.isRunning.Load(),
		"activeJobs":    w.activeJobs.Load(),
		"processed":     w.processed.Load(),
		"errors":        w.errors.Load(),
		"maxConcurrent": w.maxConcurrent,
	}
}

// ID returns the worker's unique identifier
func (w *Worker) ID() string {
	return w.id
}

// processLoop runs in its own goroutine and continuously processes ads
func (w *Worker) processLoop(ctx context.Context, workerNum int) {
	defer w.wg.Done()

	// Create a done channel that's closed when the context is done
	doneCh := ctx.Done()

	for {
		// Check if we should shut down
		select {
		case <-doneCh:
			return
		case <-w.shutdownCh:
			return
		default:
		}

		// Try to get an ad from the queue
		ad, err := w.queue.Dequeue(ctx)
		if err != nil {
			if err == ErrQueueEmpty {
				// Queue is empty, wait a bit before trying again
				select {
				case <-time.After(500 * time.Millisecond):
					continue
				case <-doneCh:
					return
				case <-w.shutdownCh:
					return
				}
			} else if err == ErrItemLocked {
				// Item is locked by another worker, try again immediately
				continue
			} else {
				// Other error, increment error counter
				w.errors.Add(1)

				// Wait a bit before trying again
				select {
				case <-time.After(1 * time.Second):
					continue
				case <-doneCh:
					return
				case <-w.shutdownCh:
					return
				}
			}
		}

		// We got an ad to process
		w.activeJobs.Add(1)

		// Process the ad (in a real system, this might involve sending to another service)
		analysis, err := w.processAd(ctx, ad)

		// Mark job as no longer active
		w.activeJobs.Add(-1)

		if err != nil {
			// Processing failed
			w.errors.Add(1)
			// In a real system, we would handle this more gracefully
			continue
		}

		// Processing succeeded, update the ad with the analysis results
		err = w.queue.Complete(ctx, ad, analysis)
		if err != nil {
			// Failed to update the ad
			w.errors.Add(1)
			continue
		}

		// Successfully processed an ad
		w.processed.Add(1)
	}
}

// processAd performs the actual analysis of an ad
// In a real system, this might involve machine learning models or complex logic
func (w *Worker) processAd(ctx context.Context, ad *models.Ad) (models.AdAnalysis, error) {
	// Simulate processing time with some randomness
	variance := float64(w.processTime) * 0.3 // 30% variance
	processingTime := time.Duration(float64(w.processTime) + (rand.Float64() * variance) - variance/2)

	// Check if we should cancel processing
	select {
	case <-ctx.Done():
		return models.AdAnalysis{}, ctx.Err()
	case <-w.shutdownCh:
		return models.AdAnalysis{}, fmt.Errorf("worker shutting down")
	case <-time.After(processingTime):
		// Processing completed
	}

	// Use the analyzer to analyze the ad
	return w.analyzer.Analyze(ad)
}

// WorkerPool manages a collection of workers
type WorkerPool struct {
	queue         *DistributedPriorityQueue
	analyzer      *Analyzer
	workerOptions WorkerOptions
	workers       map[string]*Worker
	mu            sync.RWMutex

	// Auto-scaling options
	enableAutoScale bool
	minWorkers      int
	maxWorkers      int
	targetQueueSize int

	// For shutdown signaling
	shutdownCh chan struct{}
}

// WorkerPoolOptions configures the worker pool
type WorkerPoolOptions struct {
	// Worker options for each worker
	WorkerOptions WorkerOptions

	// Minimum number of workers
	MinWorkers int

	// Maximum number of workers
	MaxWorkers int

	// Enable auto-scaling
	EnableAutoScale bool

	// Target queue size for auto-scaling
	TargetQueueSize int
}

// DefaultWorkerPoolOptions returns sensible defaults for worker pool options
func DefaultWorkerPoolOptions() WorkerPoolOptions {
	return WorkerPoolOptions{
		WorkerOptions:   DefaultWorkerOptions(),
		MinWorkers:      2,
		MaxWorkers:      10,
		EnableAutoScale: true,
		TargetQueueSize: 100,
	}
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(
	queue *DistributedPriorityQueue,
	analyzer *Analyzer,
	options WorkerPoolOptions,
) *WorkerPool {
	return &WorkerPool{
		queue:           queue,
		analyzer:        analyzer,
		workerOptions:   options.WorkerOptions,
		workers:         make(map[string]*Worker),
		enableAutoScale: options.EnableAutoScale,
		minWorkers:      options.MinWorkers,
		maxWorkers:      options.MaxWorkers,
		targetQueueSize: options.TargetQueueSize,
		shutdownCh:      make(chan struct{}),
	}
}

// Start launches the worker pool with the initial number of workers
func (wp *WorkerPool) Start(ctx context.Context) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	// Start with minimum number of workers
	for i := 0; i < wp.minWorkers; i++ {
		worker := NewWorker(wp.queue, wp.analyzer, wp.workerOptions)
		err := worker.Start(ctx)
		if err != nil {
			// Cleanup any workers we started before the error
			for _, w := range wp.workers {
				w.Stop(false)
			}
			return fmt.Errorf("failed to start worker: %w", err)
		}

		wp.workers[worker.ID()] = worker
	}

	// If auto-scaling is enabled, start the auto-scaler
	if wp.enableAutoScale {
		go wp.autoScalerLoop(ctx)
	}

	return nil
}

// Stop gracefully shuts down all workers
func (wp *WorkerPool) Stop(waitForCompletion bool) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	// Signal the auto-scaler to stop
	close(wp.shutdownCh)

	// Stop all workers
	for _, worker := range wp.workers {
		worker.Stop(waitForCompletion)
	}

	// Clear the workers map
	if !waitForCompletion {
		wp.workers = make(map[string]*Worker)
	}

	return nil
}

// AddWorker adds a new worker to the pool
func (wp *WorkerPool) AddWorker(ctx context.Context) (*Worker, error) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if len(wp.workers) >= wp.maxWorkers {
		return nil, fmt.Errorf("maximum worker count reached")
	}

	worker := NewWorker(wp.queue, wp.analyzer, wp.workerOptions)
	err := worker.Start(ctx)
	if err != nil {
		return nil, err
	}

	wp.workers[worker.ID()] = worker
	return worker, nil
}

// RemoveWorker removes a worker from the pool
func (wp *WorkerPool) RemoveWorker(workerID string, waitForCompletion bool) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	worker, exists := wp.workers[workerID]
	if !exists {
		return fmt.Errorf("worker not found: %s", workerID)
	}

	if len(wp.workers) <= wp.minWorkers {
		return fmt.Errorf("minimum worker count reached")
	}

	err := worker.Stop(waitForCompletion)
	if err != nil {
		return err
	}

	if !waitForCompletion {
		delete(wp.workers, workerID)
	}

	return nil
}

// GetStats returns statistics for all workers
func (wp *WorkerPool) GetStats() map[string]interface{} {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	workerStats := make(map[string]interface{})
	totalProcessed := int64(0)
	totalErrors := int64(0)
	totalActive := int32(0)

	for id, worker := range wp.workers {
		stats := worker.Stats()
		workerStats[id] = stats

		totalProcessed += stats["processed"].(int64)
		totalErrors += stats["errors"].(int64)
		totalActive += stats["activeJobs"].(int32)
	}

	return map[string]interface{}{
		"workerCount":    len(wp.workers),
		"totalProcessed": totalProcessed,
		"totalErrors":    totalErrors,
		"totalActive":    totalActive,
		"workers":        workerStats,
	}
}

// autoScalerLoop adjusts the number of workers based on queue size
func (wp *WorkerPool) autoScalerLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-wp.shutdownCh:
			return
		case <-ticker.C:
			// Check queue size
			queueSize, err := wp.queue.Size(ctx)
			if err != nil {
				// Log error in real implementation
				continue
			}

			wp.mu.Lock()

			currentWorkers := len(wp.workers)

			// Scale up if queue is too large
			if queueSize > wp.targetQueueSize && currentWorkers < wp.maxWorkers {
				// Add a worker
				worker := NewWorker(wp.queue, wp.analyzer, wp.workerOptions)
				err := worker.Start(ctx)
				if err == nil {
					wp.workers[worker.ID()] = worker
				}
			}

			// Scale down if queue is too small and we have more than minimum workers
			if queueSize < wp.targetQueueSize/2 && currentWorkers > wp.minWorkers {
				// Find a worker to remove
				var workerToRemove string
				for id, worker := range wp.workers {
					if worker.activeJobs.Load() == 0 {
						workerToRemove = id
						break
					}
				}

				if workerToRemove != "" {
					wp.workers[workerToRemove].Stop(false)
					delete(wp.workers, workerToRemove)
				}
			}

			wp.mu.Unlock()
		}
	}
}
