package service

import (
	"context"
	"fmt"
	"time"

	"github.com/nghiack7/game-ad-service/internal/domain/models"
	"github.com/nghiack7/game-ad-service/internal/interfaces/dtos"
	"github.com/nghiack7/game-ad-service/internal/interfaces/mapper"
	"github.com/nghiack7/game-ad-service/internal/processor"
	"github.com/nghiack7/game-ad-service/internal/storage"
	"github.com/nghiack7/game-ad-service/pkg/errors"
	"github.com/nghiack7/game-ad-service/pkg/utils"
)

// AdService handles ad-related business logic
type AdService interface {
	// SubmitAd submits a new ad for processing
	SubmitAd(ctx context.Context, ad *models.Ad) (*dtos.AdDTO, error)

	// GetAd retrieves an ad by ID
	GetAd(ctx context.Context, adID string) (*dtos.AdDTO, error)

	// ListAds lists ads with optional status filtering
	ListAds(ctx context.Context, status string, limit, offset int) ([]*dtos.AdDTO, error)

	// GetStats returns processing statistics
	GetStats(ctx context.Context) (map[string]interface{}, error)
}

// DefaultAdService is the default implementation of AdService
type DefaultAdService struct {
	repo       storage.Repository
	cacheRepo  storage.CacheRepository
	queue      *processor.DistributedPriorityQueue
	workerPool *processor.WorkerPool
}

var _ AdService = (*DefaultAdService)(nil)

// NewAdService creates a new ad service
func NewAdService(
	repo storage.Repository,
	cacheRepo storage.CacheRepository,
	queue *processor.DistributedPriorityQueue,
	workerPool *processor.WorkerPool,
) *DefaultAdService {
	return &DefaultAdService{
		repo:       repo,
		cacheRepo:  cacheRepo,
		queue:      queue,
		workerPool: workerPool,
	}
}

// SubmitAd submits a new ad for processing
func (s *DefaultAdService) SubmitAd(ctx context.Context, ad *models.Ad) (*dtos.AdDTO, error) {
	// Validate ad data
	if ad == nil {
		return nil, errors.New(errors.ErrInvalidRequest)
	}

	// Generate ID if not provided
	if ad.AdID == "" {
		ad.AdID = utils.GenerateID()
	}

	// Set initial metadata
	ad.CreatedAt = time.Now().UTC()
	ad.Version = 1
	ad.Status = models.StatusSubmitted
	ad.Attempts = 0

	// Save to repository
	_, err := s.repo.Save(ctx, ad)
	if err != nil {
		return nil, fmt.Errorf("failed to save ad: %w", err)
	}

	// Enqueue for processing
	// Use a background context for queue operations to avoid cancellation
	// from the HTTP request context
	bgCtx := context.Background()

	// Add to processing queue with priority
	err = s.queue.Enqueue(bgCtx, ad)
	if err != nil {
		// Log the error but don't fail the request
		// The ad is saved and can be processed later
		// In a real implementation, we would use a proper logger
		fmt.Printf("Warning: Failed to enqueue ad %s: %v\n", ad.AdID, err)
	}

	// Return the created ad
	return mapper.MapAdToAdDTO(ad), nil
}

// GetAd retrieves an ad by ID
func (s *DefaultAdService) GetAd(ctx context.Context, adID string) (*dtos.AdDTO, error) {
	// Validate input
	if adID == "" {
		return nil, errors.New(errors.ErrInvalidRequest)
	}

	// Try to get from cache first for better performance
	ad, err := s.cacheRepo.Get(ctx, adID)
	if err == nil && ad != nil {
		return mapper.MapAdToAdDTO(ad), nil
	}

	// If not in cache, get from main repository
	ad, err = s.repo.Get(ctx, adID)
	if err != nil {
		if err == storage.ErrAdNotFound {
			return nil, errors.New(errors.ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get ad: %w", err)
	}

	// Cache the result for future requests
	// Use a background context for cache operations to avoid cancellation
	bgCtx := context.Background()
	s.cacheRepo.Set(bgCtx, adID, ad, 300) // Cache for 5 minutes

	return mapper.MapAdToAdDTO(ad), nil
}

// ListAds lists ads with optional status filtering
func (s *DefaultAdService) ListAds(ctx context.Context, status string, limit, offset int) ([]*dtos.AdDTO, error) {
	// Apply reasonable limits
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	if offset < 0 {
		offset = 0
	}

	var ads []*models.Ad
	var err error

	if status != "" {
		// Convert string to AdStatus
		adStatus := models.AdStatus(status)
		ads, err = s.repo.FindByStatus(ctx, adStatus, limit, offset)
	} else {
		// Get all ads (paginated)
		// This is a simplification - in reality we might need a specialized method
		// to get all ads efficiently
		allStatuses := []models.AdStatus{
			models.StatusSubmitted,
			models.StatusQueued,
			models.StatusProcessing,
			models.StatusCompleted,
			models.StatusFailed,
		}

		for _, st := range allStatuses {
			statusAds, err := s.repo.FindByStatus(ctx, st, limit, offset)
			if err != nil {
				continue
			}
			ads = append(ads, statusAds...)

			// Stop if we have enough
			if len(ads) >= limit {
				ads = ads[:limit]
				break
			}
		}
	}

	if err != nil {
		return nil, errors.New(errors.ErrInternalServer)
	}
	var adDTOs []*dtos.AdDTO
	for _, ad := range ads {
		adDTOs = append(adDTOs, mapper.MapAdToAdDTO(ad))
	}
	return adDTOs, nil
}

// GetStats returns processing statistics
func (s *DefaultAdService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get queue stats
	queueSize, _ := s.queue.Size(ctx)
	stats["queueSize"] = queueSize

	// Get worker stats
	if s.workerPool != nil {
		workerStats := s.workerPool.GetStats()
		stats["workers"] = workerStats
	}

	// Get counts by status
	statusCounts := make(map[string]int)
	for _, status := range []models.AdStatus{
		models.StatusSubmitted,
		models.StatusQueued,
		models.StatusProcessing,
		models.StatusCompleted,
		models.StatusFailed,
	} {
		count, _ := s.repo.CountByStatus(ctx, status)
		statusCounts[string(status)] = count
	}

	stats["statusCounts"] = statusCounts
	stats["timestamp"] = time.Now().UTC().Format(time.RFC3339)

	return stats, nil
}
