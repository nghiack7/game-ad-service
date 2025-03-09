package storage

import (
	"context"
	"errors"
	"time"

	"github.com/nghiack7/game-ad-service/internal/domain/models"
)

// Common errors
var (
	// ErrAdNotFound is returned when an ad cannot be found
	ErrAdNotFound = errors.New("ad not found")

	// ErrConcurrentModification is returned when there's a version conflict during update
	ErrConcurrentModification = errors.New("concurrent modification detected")

	// ErrInvalidAdData is returned when ad data is invalid
	ErrInvalidAdData = errors.New("invalid ad data")
)

// Repository defines the interface for ad storage operations
type Repository interface {
	// Save stores a new ad and returns its ID
	Save(ctx context.Context, ad *models.Ad) (string, error)

	// Get retrieves an ad by ID
	Get(ctx context.Context, adID string) (*models.Ad, error)

	// Update updates an existing ad
	// Returns ErrConcurrentModification if version doesn't match
	Update(ctx context.Context, ad *models.Ad) error

	// FindByStatus finds all ads with the given status
	FindByStatus(ctx context.Context, status models.AdStatus, limit, offset int) ([]*models.Ad, error)

	// FindByPriority finds ads with given status ordered by priority (highest first)
	FindByPriority(ctx context.Context, status models.AdStatus, limit int) ([]*models.Ad, error)

	// CountByStatus counts ads with the given status
	CountByStatus(ctx context.Context, status models.AdStatus) (int, error)

	// DeleteOlderThan deletes ads older than the given timestamp
	DeleteOlderThan(ctx context.Context, days int) (int, error)

	// AssignToWorker marks ads as assigned to a specific worker
	// This is used to prevent multiple workers from processing the same ad
	AssignToWorker(ctx context.Context, adIDs []string, workerID string) (int, error)

	// ReleaseWorkerAssignments releases all ads assigned to a worker
	// This is used when a worker becomes unavailable
	ReleaseWorkerAssignments(ctx context.Context, workerID string) (int, error)

	// Atomic status transaction operations

	// AtomicStatusUpdate updates ad status if current version matches
	AtomicStatusUpdate(ctx context.Context, adID string, currentVersion int64,
		fromStatus, toStatus models.AdStatus) error

	// BatchUpdateStatus updates multiple ads' status in a single operation
	BatchUpdateStatus(ctx context.Context, adIDs []string, toStatus models.AdStatus) (int, error)

	// For distributed consensus

	// AcquireLock attempts to acquire a distributed lock on an ad
	// Returns true if successful, false if already locked
	AcquireLock(ctx context.Context, adID, ownerID string, ttlSeconds time.Duration) (bool, error)

	// ReleaseLock releases a distributed lock if owned by ownerID
	ReleaseLock(ctx context.Context, adID, ownerID string) (bool, error)
}

// CacheRepository defines operations for a faster, distributed cache layer
type CacheRepository interface {
	// Cache operations with TTL
	Set(ctx context.Context, adID string, ad *models.Ad, ttlSeconds time.Duration) error
	Get(ctx context.Context, adID string) (*models.Ad, error)
	Delete(ctx context.Context, adID string) error

	// Queue operations for distributed processing
	PushToQueue(ctx context.Context, queueName string, adID string, priority int) error
	PopFromQueue(ctx context.Context, queueName string) (string, error)

	// Distributed counters and gauges
	IncrementCounter(ctx context.Context, key string) (int64, error)
	GetGauge(ctx context.Context, key string) (int64, error)
	SetGauge(ctx context.Context, key string, value int64) error
}

// RepositoryFactory creates storage implementations based on configuration
type RepositoryFactory interface {
	// CreateRepository creates a main repository implementation
	CreateRepository(ctx context.Context) (Repository, error)

	// CreateCacheRepository creates a cache repository implementation
	CreateCacheRepository(ctx context.Context) (CacheRepository, error)
}
