package processor

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/nghiack7/game-ad-service/internal/domain/models"
	"github.com/nghiack7/game-ad-service/internal/storage"
	"github.com/nghiack7/game-ad-service/pkg/logger"
)

// Common errors
var (
	// ErrQueueEmpty is returned when the queue is empty
	ErrQueueEmpty = errors.New("queue is empty")

	// ErrQueueFull is returned when the queue is at capacity
	ErrQueueFull = errors.New("queue is at capacity")

	// ErrQueueClosed is returned when operations are performed on a closed queue
	ErrQueueClosed = errors.New("queue is closed")

	// ErrItemLocked is returned when an item is already being processed
	ErrItemLocked = errors.New("item is locked by another worker")
)

// QueueOptions defines configuration options for the priority queue
type QueueOptions struct {
	// MaxCapacity defines the maximum number of items in the queue (0 = unlimited)
	MaxCapacity int

	// ProcessingTimeout defines how long an item can be in processing state
	// before it's considered failed and returned to the queue
	ProcessingTimeout time.Duration

	// LockTTL defines the TTL for distributed locks in seconds
	LockTTL time.Duration

	// MaxAttempts defines how many times an item can be processed before being marked as failed
	MaxAttempts int

	// EnableAging enables priority aging to prevent starvation of low-priority items
	EnableAging bool

	// AgingRate defines how quickly low-priority items age (higher = faster)
	AgingRate float64

	// MaxWaitTime defines the maximum time an item can wait in the queue
	// before its priority is boosted to maximum
	MaxWaitTime time.Duration
}

// DefaultQueueOptions returns sensible default options
func DefaultQueueOptions() QueueOptions {
	return QueueOptions{
		MaxCapacity:       10000,
		ProcessingTimeout: 5 * time.Minute,
		LockTTL:           30 * time.Second,
		MaxAttempts:       3,
		EnableAging:       true,
		AgingRate:         0.5,             // Increased from 0.1 for faster aging
		MaxWaitTime:       5 * time.Minute, // Reduced from 10 minutes to ensure faster processing of waiting items
	}
}

// DistributedPriorityQueue implements a scalable, distributed priority queue
// using a storage backend for coordination between multiple workers
type DistributedPriorityQueue struct {
	repo        storage.Repository
	cacheRepo   storage.CacheRepository
	instanceID  string
	options     QueueOptions
	closed      bool
	mu          sync.RWMutex
	queuedItems int64
	batchSize   int

	// For shutdown signaling
	shutdownCh chan struct{}
}

// NewDistributedPriorityQueue creates a new distributed priority queue
func NewDistributedPriorityQueue(
	repo storage.Repository,
	cacheRepo storage.CacheRepository,
	instanceID string,
	options QueueOptions,
) *DistributedPriorityQueue {
	return &DistributedPriorityQueue{
		repo:       repo,
		cacheRepo:  cacheRepo,
		instanceID: instanceID,
		options:    options,
		batchSize:  10,
		shutdownCh: make(chan struct{}),
	}
}

// Enqueue adds an ad to the queue with the specified priority
func (q *DistributedPriorityQueue) Enqueue(ctx context.Context, ad *models.Ad) error {
	q.mu.RLock()
	if q.closed {
		q.mu.RUnlock()
		return ErrQueueClosed
	}
	q.mu.RUnlock()

	// First check if we're at capacity
	if q.options.MaxCapacity > 0 {
		count, err := q.repo.CountByStatus(ctx, models.StatusQueued)
		if err != nil {
			return err
		}

		if count >= q.options.MaxCapacity {
			return ErrQueueFull
		}
	}

	// Ensure the ad has the right status
	ad.SetStatus(models.StatusQueued)

	// Store the updated ad
	err := q.repo.Update(ctx, ad)
	if err != nil {
		return err
	}

	// Add to the distributed queue for faster dequeuing
	// Use effective priority for queue ordering
	err = q.cacheRepo.PushToQueue(ctx, "ads:queue", ad.AdID, ad.EffectivePriority)
	if err != nil {
		logger.Errorf("Failed to add ad %s to queue: %v", ad.AdID, err)
	}

	// Update queue count metrics
	q.cacheRepo.IncrementCounter(ctx, "queue:count")

	return nil
}

// Dequeue retrieves the highest priority ad from the queue
func (q *DistributedPriorityQueue) Dequeue(ctx context.Context) (*models.Ad, error) {
	q.mu.RLock()
	if q.closed {
		q.mu.RUnlock()
		return nil, ErrQueueClosed
	}
	q.mu.RUnlock()

	// Try to get an item from the cache queue first (faster path)
	adID, err := q.cacheRepo.PopFromQueue(ctx, "ads:queue")
	if err == nil && adID != "" {
		// We got an item from the cache queue
		ad, err := q.repo.Get(ctx, adID)
		if err != nil {
			if errors.Is(err, storage.ErrAdNotFound) {
				// The ad was removed from the database but not the queue
				// Just try again
				return q.Dequeue(ctx)
			}
			return nil, err
		}

		// Check if this ad is actually processable
		if !ad.IsProcessable() {
			// The ad is in the wrong state, ignore it and try again
			return q.Dequeue(ctx)
		}

		// Try to acquire a lock on this ad
		locked, err := q.repo.AcquireLock(ctx, ad.AdID, q.instanceID, q.options.LockTTL)
		if err != nil {
			return nil, err
		}

		if !locked {
			// Another worker is processing this ad, skip it
			return q.Dequeue(ctx)
		}
		defer q.repo.ReleaseLock(ctx, ad.AdID, q.instanceID)

		// Important: Get the current version from the repository before making changes
		currentVersion := ad.Version

		// Use AtomicStatusUpdate to change the status in the repository
		// This ensures we're checking against the version in the repository
		err = q.repo.AtomicStatusUpdate(ctx, ad.AdID, currentVersion, models.StatusQueued, models.StatusProcessing)
		if err != nil {
			if errors.Is(err, storage.ErrConcurrentModification) {
				// Someone else modified the ad, try again
				return q.Dequeue(ctx)
			}

			return nil, err
		}

		// Now get the updated ad with the new version from the repository
		updatedAd, err := q.repo.Get(ctx, ad.AdID)
		if err != nil {
			// Release the lock since we couldn't get the updated ad
			q.repo.ReleaseLock(ctx, ad.AdID, q.instanceID)
			return nil, err
		}

		// Now we can safely modify the ad without version conflicts
		// because we're working with the latest version from the repository

		// Track the current version before modifications

		// Assign worker and increment attempts
		updatedAd.AssignWorker(q.instanceID)
		updatedAd.IncrementAttempts()

		// Update the ad in the repository with the new worker ID and attempts
		err = q.repo.Update(ctx, updatedAd)
		if err != nil {
			return nil, err
		}

		// Update queue metrics
		q.cacheRepo.GetGauge(ctx, "queue:count")

		return updatedAd, nil
	}

	// Slower path: use the repository directly
	ads, err := q.repo.FindByPriority(ctx, models.StatusQueued, 1)
	if err != nil {
		return nil, err
	}

	if len(ads) == 0 {
		return nil, ErrQueueEmpty
	}

	ad := ads[0]

	// Try to acquire a lock
	locked, err := q.repo.AcquireLock(ctx, ad.AdID, q.instanceID, q.options.LockTTL)
	if err != nil {
		return nil, err
	}

	if !locked {
		// Another worker is processing this ad
		// Instead of recursively calling Dequeue, we'll return an error
		return nil, ErrItemLocked
	}

	// Store the current version before any modifications
	currentVersion := ad.Version

	// Use AtomicStatusUpdate to change the status in the repository
	err = q.repo.AtomicStatusUpdate(ctx, ad.AdID, currentVersion, models.StatusQueued, models.StatusProcessing)
	if err != nil {
		// Release the lock since we couldn't update the status
		q.repo.ReleaseLock(ctx, ad.AdID, q.instanceID)

		if errors.Is(err, storage.ErrConcurrentModification) {
			// Someone else modified the ad
			return nil, ErrItemLocked
		}

		return nil, err
	}

	// Now that the status is updated in the repository, get the fresh ad
	updatedAd, err := q.repo.Get(ctx, ad.AdID)
	if err != nil {
		// Release the lock since we couldn't get the updated ad
		q.repo.ReleaseLock(ctx, ad.AdID, q.instanceID)
		return nil, err
	}

	// Assign the worker ID
	updatedAd.AssignWorker(q.instanceID)
	updatedAd.IncrementAttempts()

	// Update the ad with the worker ID and incremented attempts
	err = q.repo.Update(ctx, updatedAd)
	if err != nil {
		// Release the lock since we couldn't update the ad
		q.repo.ReleaseLock(ctx, ad.AdID, q.instanceID)
		return nil, err
	}

	// Update queue metrics
	q.cacheRepo.GetGauge(ctx, "queue:count")

	return updatedAd, nil
}

// Complete marks an ad as completed
func (q *DistributedPriorityQueue) Complete(ctx context.Context, ad *models.Ad, analysis models.AdAnalysis) error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	// Make sure we own the lock
	locked, err := q.repo.AcquireLock(ctx, ad.AdID, q.instanceID, q.options.LockTTL)
	if err != nil {
		return err
	}
	if !locked {
		// We don't own the lock anymore
		return ErrItemLocked
	}
	defer q.repo.ReleaseLock(ctx, ad.AdID, q.instanceID)

	// Store the current version before any modifications
	currentVersion := ad.Version

	// Use AtomicStatusUpdate to change the status in the repository
	err = q.repo.AtomicStatusUpdate(ctx, ad.AdID, currentVersion, models.StatusProcessing, models.StatusCompleted)
	if err != nil {
		return err
	}

	// Now that the status is updated in the repository, get the fresh ad
	updatedAd, err := q.repo.Get(ctx, ad.AdID)
	if err != nil {
		// Release the lock since we couldn't get the updated ad
		return err
	}

	// Set the analysis results
	updatedAd.SetAnalysisResults(analysis)

	// Update in the repository
	err = q.repo.Update(ctx, updatedAd)
	if err != nil {
		// Release the lock since we couldn't update the ad
		return err
	}
	logger.Infof("Completed ad %s", ad.AdID)

	return nil
}

// RequeueExpired checks for ads that have been in processing state for too long
// and returns them to the queue
func (q *DistributedPriorityQueue) RequeueExpired(ctx context.Context) (int, error) {
	q.mu.RLock()
	if q.closed {
		q.mu.RUnlock()
		return 0, ErrQueueClosed
	}
	q.mu.RUnlock()

	// Get ads that have been processing for too long
	expirationCutoff := time.Now().Add(-q.options.ProcessingTimeout)

	// In a real implementation, we would have a more efficient way to query by status and modification time
	// For simplicity, we'll just get all processing ads
	ads, err := q.repo.FindByStatus(ctx, models.StatusProcessing, 100, 0)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, ad := range ads {
		// Check if this ad has been processing for too long
		// Use the ProcessingStartedAt field to determine if an ad has been processing too long
		if ad.ProcessingStartedAt != nil && ad.ProcessingStartedAt.Before(expirationCutoff) {
			// Check if we should retry or mark as failed
			if ad.Attempts >= q.options.MaxAttempts {
				// Mark as failed
				ad.SetError("Exceeded maximum processing attempts")
				err := q.repo.Update(ctx, ad)
				if err != nil {
					// Just log and continue in a real implementation
					continue
				}
			} else {
				// Store the current version before modifying the ad
				currentVersion := ad.Version

				// Use AtomicStatusUpdate directly without modifying the ad first
				// This avoids the version mismatch issue
				err = q.repo.AtomicStatusUpdate(ctx, ad.AdID, currentVersion, models.StatusProcessing, models.StatusQueued)
				if err != nil {
					// Just log and continue in a real implementation
					continue
				}

				// Refresh the ad from the repository to get the updated version
				updatedAd, err := q.repo.Get(ctx, ad.AdID)
				if err != nil {
					// Just log and continue in a real implementation
					continue
				}

				// Add back to the cache queue with the effective priority
				// Use the priority from the updated ad
				err = q.cacheRepo.PushToQueue(ctx, "ads:queue", updatedAd.AdID, updatedAd.Priority)
				if err != nil {
					// Log error and continue
					continue
				}

				count++
			}
		}
	}

	return count, nil
}

// Size returns the approximate number of items in the queue
func (q *DistributedPriorityQueue) Size(ctx context.Context) (int, error) {
	return q.repo.CountByStatus(ctx, models.StatusQueued)
}

// Close shuts down the queue
func (q *DistributedPriorityQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return nil
	}

	q.closed = true
	close(q.shutdownCh)

	return nil
}

// StartMaintenanceLoop starts a maintenance loop that periodically requeues expired items
// and performs other maintenance tasks
func (q *DistributedPriorityQueue) StartMaintenanceLoop(ctx context.Context) {
	// Run maintenance tasks every minute
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// Run priority aging more frequently (every 10 seconds)
	agingTicker := time.NewTicker(10 * time.Second)
	defer agingTicker.Stop()

	// Run priority boosting every 30 seconds
	boostTicker := time.NewTicker(30 * time.Second)
	defer boostTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-q.shutdownCh:
			return
		case <-ticker.C:
			// Requeue expired items
			requeueCount, err := q.RequeueExpired(ctx)
			if err != nil {
				// Log error in real implementation
				continue
			}

			if requeueCount > 0 {
				// Log requeue count in real implementation
			}

			// Clean up expired locks
			// Update metrics
		case <-agingTicker.C:
			// Only run aging if enabled
			if !q.options.EnableAging {
				continue
			}

			// Update priorities of queued items to prevent starvation
			count, err := q.updateQueuePriorities(ctx)
			if err != nil {
				// Log error in real implementation
				continue
			}

			if count > 0 {
				// Log priority updates in real implementation
			}
		case <-boostTicker.C:
			// Boost priority of items that have been waiting too long
			count, err := q.BoostLowPriorityItems(ctx)
			if err != nil {
				// Log error in real implementation
				continue
			}

			if count > 0 {
				// Log boosted items in real implementation
			}
		}
	}
}

// updateQueuePriorities updates the effective priorities of queued items based on waiting time
func (q *DistributedPriorityQueue) updateQueuePriorities(ctx context.Context) (int, error) {
	// Get all queued ads
	ads, err := q.repo.FindByStatus(ctx, models.StatusQueued, 100, 0)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, ad := range ads {
		// Skip if no QueuedAt timestamp
		if ad.QueuedAt == nil {
			continue
		}

		// Store the current version and effective priority
		currentVersion := ad.Version
		originalPriority := ad.EffectivePriority

		// Calculate new effective priority based on waiting time
		// This updates ad.EffectivePriority internally
		newPriority := ad.CalculateEffectivePriority(q.options.AgingRate, q.options.MaxWaitTime)

		// If priority changed, update the ad and requeue with new priority
		if newPriority != originalPriority {
			// Update the ad in the repository
			// We need to be careful not to overwrite other changes
			// Get the latest version from the repository
			latestAd, err := q.repo.Get(ctx, ad.AdID)
			if err != nil {
				// Log error and continue with next ad
				continue
			}

			// Only update if the version hasn't changed
			if latestAd.Version == currentVersion {
				// Update the effective priority
				latestAd.EffectivePriority = newPriority

				// Update in the repository
				err = q.repo.Update(ctx, latestAd)
				if err != nil {
					// Log error and continue with next ad
					continue
				}

				// Remove from queue and re-add with new priority
				// In a real implementation, we would have an updatePriority operation
				// For now, we'll just push again (which might create duplicates in some implementations)
				err = q.cacheRepo.PushToQueue(ctx, "ads:queue", latestAd.AdID, latestAd.EffectivePriority)
				if err != nil {
					// Log error and continue
					continue
				}

				count++
			}
		}
	}

	return count, nil
}

// BoostLowPriorityItems boosts the priority of items that have been waiting too long
func (q *DistributedPriorityQueue) BoostLowPriorityItems(ctx context.Context) (int, error) {
	q.mu.RLock()
	if q.closed {
		q.mu.RUnlock()
		return 0, ErrQueueClosed
	}
	q.mu.RUnlock()

	// Get all queued ads
	ads, err := q.repo.FindByStatus(ctx, models.StatusQueued, 100, 0)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, ad := range ads {
		// Skip if no QueuedAt timestamp
		if ad.QueuedAt == nil {
			continue
		}

		// Check if this ad has been waiting longer than MaxWaitTime
		if ad.HasExceededMaxWaitTime(q.options.MaxWaitTime) {
			// Store the current version
			currentVersion := ad.Version

			// Get the latest version from the repository
			latestAd, err := q.repo.Get(ctx, ad.AdID)
			if err != nil {
				// Log error and continue with next ad
				continue
			}

			// Only update if the version hasn't changed
			if latestAd.Version == currentVersion {
				// Boost to highest priority (1)
				// Store old priority for logging in a real implementation
				// oldPriority := latestAd.EffectivePriority
				latestAd.EffectivePriority = 1

				// Update the ad in the repository
				err := q.repo.Update(ctx, latestAd)
				if err != nil {
					// Log error and continue with next ad
					continue
				}

				// Remove from queue and re-add with new priority
				err = q.cacheRepo.PushToQueue(ctx, "ads:queue", latestAd.AdID, latestAd.EffectivePriority)
				if err != nil {
					// Log error and continue
					continue
				}

				count++
			}
		}
	}

	return count, nil
}
