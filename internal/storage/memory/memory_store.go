package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/nghiack7/game-ad-service/internal/domain/models"
	"github.com/nghiack7/game-ad-service/internal/storage"
)

// MemoryRepository implements the Repository interface with in-memory storage
type MemoryRepository struct {
	// Mutex for thread safety
	mu sync.RWMutex

	// Map of ads by ID
	ads map[string]*models.Ad

	// Lock map for distributed locking simulation
	locks map[string]lockInfo
}

// lockInfo stores information about a lock
type lockInfo struct {
	ownerID   string
	expiresAt time.Time
}

// NewMemoryRepository creates a new in-memory repository
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		ads:   make(map[string]*models.Ad),
		locks: make(map[string]lockInfo),
	}
}

// Save stores a new ad
func (m *MemoryRepository) Save(ctx context.Context, ad *models.Ad) (string, error) {
	if ad == nil {
		return "", fmt.Errorf("%w: ad is nil", storage.ErrInvalidAdData)
	}

	if ad.AdID == "" {
		return "", fmt.Errorf("%w: ad ID is required", storage.ErrInvalidAdData)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Store a deep copy to prevent external mutations
	adCopy := *ad
	m.ads[ad.AdID] = &adCopy

	return ad.AdID, nil
}

// Get retrieves an ad by ID
func (m *MemoryRepository) Get(ctx context.Context, adID string) (*models.Ad, error) {
	if adID == "" {
		return nil, fmt.Errorf("%w: ad ID is required", storage.ErrInvalidAdData)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	ad, exists := m.ads[adID]
	if !exists {
		return nil, fmt.Errorf("%w: ad with ID %s not found", storage.ErrAdNotFound, adID)
	}

	// Return a copy to prevent external mutations
	adCopy := *ad
	return &adCopy, nil
}

// Update updates an existing ad
func (m *MemoryRepository) Update(ctx context.Context, ad *models.Ad) error {
	if ad == nil {
		return fmt.Errorf("%w: ad is nil", storage.ErrInvalidAdData)
	}

	if ad.AdID == "" {
		return fmt.Errorf("%w: ad ID is required", storage.ErrInvalidAdData)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.ads[ad.AdID]
	if !exists {
		return fmt.Errorf("%w: ad with ID %s not found", storage.ErrAdNotFound, ad.AdID)
	}

	// Check if the version has changed
	if existing.Version != ad.Version {
		return fmt.Errorf("%w: ad with ID %s has been modified", storage.ErrConcurrentModification, ad.AdID)
	}

	// Increment the version when updating
	ad.Version++

	// Store a copy
	adCopy := *ad
	m.ads[ad.AdID] = &adCopy

	return nil
}

// FindByStatus finds all ads with the given status
func (m *MemoryRepository) FindByStatus(ctx context.Context, status models.AdStatus, limit, offset int) ([]*models.Ad, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*models.Ad

	// First, collect all matching ads
	for _, ad := range m.ads {
		if ad.Status == status {
			// Create a copy
			adCopy := *ad
			result = append(result, &adCopy)
		}
	}

	// Sort by creation time (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	// Apply pagination
	if offset >= len(result) {
		return []*models.Ad{}, nil
	}

	end := offset + limit
	if end > len(result) {
		end = len(result)
	}

	return result[offset:end], nil
}

// FindByPriority finds ads with given status ordered by priority
func (m *MemoryRepository) FindByPriority(ctx context.Context, status models.AdStatus, limit int) ([]*models.Ad, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*models.Ad

	// Collect all matching ads
	for _, ad := range m.ads {
		if ad.Status == status {
			// Create a copy
			adCopy := *ad
			result = append(result, &adCopy)
		}
	}

	// Sort by effective priority (lower number = higher priority) and then by creation time
	sort.Slice(result, func(i, j int) bool {
		// First compare by effective priority if available
		if result[i].EffectivePriority != result[j].EffectivePriority {
			return result[i].EffectivePriority < result[j].EffectivePriority
		}

		// If effective priorities are the same, compare by original priority
		if result[i].Priority != result[j].Priority {
			return result[i].Priority < result[j].Priority
		}

		// If priorities are the same, older items come first (FIFO within same priority)
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})

	// Apply limit
	if limit > len(result) {
		limit = len(result)
	}

	return result[:limit], nil
}

// CountByStatus counts ads with the given status
func (m *MemoryRepository) CountByStatus(ctx context.Context, status models.AdStatus) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, ad := range m.ads {
		if ad.Status == status {
			count++
		}
	}

	return count, nil
}

// DeleteOlderThan deletes ads older than the given number of days
func (m *MemoryRepository) DeleteOlderThan(ctx context.Context, days int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -days)
	count := 0

	for id, ad := range m.ads {
		if ad.CreatedAt.Before(cutoff) {
			delete(m.ads, id)
			count++
		}
	}

	return count, nil
}

// AssignToWorker marks ads as assigned to a specific worker
func (m *MemoryRepository) AssignToWorker(ctx context.Context, adIDs []string, workerID string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for _, id := range adIDs {
		if ad, exists := m.ads[id]; exists {
			ad.WorkerID = workerID
			ad.Version++
			count++
		}
	}

	return count, nil
}

// ReleaseWorkerAssignments releases all ads assigned to a worker
func (m *MemoryRepository) ReleaseWorkerAssignments(ctx context.Context, workerID string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for _, ad := range m.ads {
		if ad.WorkerID == workerID {
			ad.WorkerID = ""
			ad.Version++
			count++
		}
	}

	return count, nil
}

// AtomicStatusUpdate updates ad status if current version matches
func (m *MemoryRepository) AtomicStatusUpdate(ctx context.Context, adID string, currentVersion int64,
	fromStatus, toStatus models.AdStatus) error {

	m.mu.Lock()
	defer m.mu.Unlock()

	ad, exists := m.ads[adID]
	if !exists {
		return storage.ErrAdNotFound
	}

	if ad.Version != currentVersion {
		return storage.ErrConcurrentModification
	}

	if ad.Status != fromStatus {
		return storage.ErrConcurrentModification
	}

	ad.Status = toStatus
	ad.Version++

	// Handle completed status
	if toStatus == models.StatusCompleted {
		now := time.Now().UTC()
		ad.CompletedAt = &now
	}

	return nil
}

// BatchUpdateStatus updates multiple ads' status in a single operation
func (m *MemoryRepository) BatchUpdateStatus(ctx context.Context, adIDs []string, toStatus models.AdStatus) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for _, id := range adIDs {
		if ad, exists := m.ads[id]; exists {
			ad.Status = toStatus
			ad.Version++

			// Handle completed status
			if toStatus == models.StatusCompleted {
				now := time.Now().UTC()
				ad.CompletedAt = &now
			}

			count++
		}
	}

	return count, nil
}

// AcquireLock attempts to acquire a distributed lock on an ad
func (m *MemoryRepository) AcquireLock(ctx context.Context, adID, ownerID string, ttlSeconds time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// Check if there's an existing lock
	if lock, exists := m.locks[adID]; exists {
		// If the lock has expired, we can acquire it
		if now.After(lock.expiresAt) {
			m.locks[adID] = lockInfo{
				ownerID:   ownerID,
				expiresAt: now.Add(ttlSeconds),
			}
			return true, nil
		}
		if ownerID != lock.ownerID {
			return false, nil
		}
		// extend the lock
		m.locks[adID] = lockInfo{
			ownerID:   ownerID,
			expiresAt: now.Add(ttlSeconds),
		}
		return true, nil
	}

	// No existing lock, so we can acquire it
	m.locks[adID] = lockInfo{
		ownerID:   ownerID,
		expiresAt: now.Add(ttlSeconds),
	}
	return true, nil
}

// ReleaseLock releases a distributed lock if owned by ownerID
func (m *MemoryRepository) ReleaseLock(ctx context.Context, adID, ownerID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if there's an existing lock
	if lock, exists := m.locks[adID]; exists {
		// Can only release if we own the lock
		if lock.ownerID == ownerID {
			delete(m.locks, adID)
			return true, nil
		}
		return false, nil
	}

	// No lock exists
	return true, nil
}

// Delete removes an ad from the cache
func (m *MemoryRepository) Delete(ctx context.Context, adID string) error {
	if adID == "" {
		return fmt.Errorf("%w: ad ID is required", storage.ErrInvalidAdData)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.ads[adID]; !exists {
		return fmt.Errorf("%w: ad with ID %s not found", storage.ErrAdNotFound, adID)
	}

	delete(m.ads, adID)
	return nil
}

// PushToQueue adds an ad ID to a priority queue
func (m *MemoryRepository) PushToQueue(ctx context.Context, queueName string, adID string, priority int) error {
	if adID == "" {
		return fmt.Errorf("%w: ad ID is required", storage.ErrInvalidAdData)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Verify the ad exists
	ad, exists := m.ads[adID]
	if !exists {
		return fmt.Errorf("%w: ad with ID %s not found", storage.ErrAdNotFound, adID)
	}

	// Update the effective priority
	ad.EffectivePriority = priority

	return nil
}

// PopFromQueue retrieves and removes the highest priority item from a queue
func (m *MemoryRepository) PopFromQueue(ctx context.Context, queueName string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find all queued ads
	var queuedAds []*models.Ad
	for _, ad := range m.ads {
		if ad.Status == models.StatusQueued {
			queuedAds = append(queuedAds, ad)
		}
	}

	if len(queuedAds) == 0 {
		return "", storage.ErrAdNotFound
	}

	// Sort by effective priority (lower number = higher priority)
	sort.Slice(queuedAds, func(i, j int) bool {
		if queuedAds[i].EffectivePriority != queuedAds[j].EffectivePriority {
			return queuedAds[i].EffectivePriority < queuedAds[j].EffectivePriority
		}
		return queuedAds[i].CreatedAt.Before(queuedAds[j].CreatedAt)
	})

	// Return the highest priority ad ID
	return queuedAds[0].AdID, nil
}

// IncrementCounter increments a counter and returns the new value
func (m *MemoryRepository) IncrementCounter(ctx context.Context, key string) (int64, error) {
	// Simple implementation that always returns 1
	return 1, nil
}

// GetGauge gets the value of a gauge
func (m *MemoryRepository) GetGauge(ctx context.Context, key string) (int64, error) {
	// Simple implementation that always returns 0
	return 0, nil
}

// SetGauge sets the value of a gauge
func (m *MemoryRepository) SetGauge(ctx context.Context, key string, value int64) error {
	// No-op implementation
	return nil
}

// Set stores an ad in the cache with the given TTL
func (m *MemoryRepository) Set(ctx context.Context, adID string, ad *models.Ad, ttlSeconds time.Duration) error {
	if ad == nil {
		return fmt.Errorf("%w: ad is nil", storage.ErrInvalidAdData)
	}

	if adID == "" {
		return fmt.Errorf("%w: ad ID is required", storage.ErrInvalidAdData)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Store a deep copy to prevent external mutations
	adCopy := *ad
	m.ads[adID] = &adCopy

	return nil
}
