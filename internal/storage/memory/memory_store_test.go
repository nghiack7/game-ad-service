package memory

import (
	"context"
	"testing"
	"time"

	"github.com/nghiack7/game-ad-service/internal/domain/models"
	"github.com/nghiack7/game-ad-service/internal/storage"
	"github.com/stretchr/testify/assert"
)

func TestMemoryRepository_Save(t *testing.T) {
	// Setup
	repo := NewMemoryRepository()
	ctx := context.Background()
	ad := &models.Ad{
		AdID: "test-ad-1",
		AdSubmission: models.AdSubmission{
			Title:       "Test Ad",
			Description: "Test description",
		},
		Status:    models.StatusSubmitted,
		CreatedAt: time.Now().UTC(),
		Version:   1,
	}

	// Test
	adID, err := repo.Save(ctx, ad)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, "test-ad-1", adID)

	// Verify the ad was stored
	savedAd, err := repo.Get(ctx, adID)
	assert.NoError(t, err)
	assert.Equal(t, ad.Title, savedAd.Title)
	assert.Equal(t, ad.Description, savedAd.Description)
	assert.Equal(t, ad.Status, savedAd.Status)
}

func TestMemoryRepository_Get_NotFound(t *testing.T) {
	// Setup
	repo := NewMemoryRepository()
	ctx := context.Background()

	// Test
	_, err := repo.Get(ctx, "non-existent-id")

	// Verify
	assert.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrAdNotFound)
}

func TestMemoryRepository_Update(t *testing.T) {
	// Setup
	repo := NewMemoryRepository()
	ctx := context.Background()
	ad := &models.Ad{
		AdID: "test-ad-2",
		AdSubmission: models.AdSubmission{
			Title:       "Test Ad",
			Description: "Test description",
		},
		Status:    models.StatusSubmitted,
		CreatedAt: time.Now().UTC(),
		Version:   1,
	}

	// Save the ad first
	_, err := repo.Save(ctx, ad)
	assert.NoError(t, err)

	// Update the ad
	ad.Status = models.StatusQueued
	ad.Version = 2

	// Test
	err = repo.Update(ctx, ad)

	// Verify
	assert.NoError(t, err)

	// Verify the ad was updated
	updatedAd, err := repo.Get(ctx, ad.AdID)
	assert.NoError(t, err)
	assert.Equal(t, models.StatusQueued, updatedAd.Status)
	assert.Equal(t, int64(2), updatedAd.Version)
}

func TestMemoryRepository_Update_ConcurrentModification(t *testing.T) {
	// Setup
	repo := NewMemoryRepository()
	ctx := context.Background()
	ad := &models.Ad{
		AdID: "test-ad-3",
		AdSubmission: models.AdSubmission{
			Title:       "Test Ad",
			Description: "Test description",
		},
		Status:    models.StatusSubmitted,
		CreatedAt: time.Now().UTC(),
		Version:   1,
	}

	// Save the ad first
	_, err := repo.Save(ctx, ad)
	assert.NoError(t, err)

	// Update with wrong version
	ad.Status = models.StatusQueued
	ad.Version = 3 // Should be 2

	// Test
	err = repo.Update(ctx, ad)

	// Verify
	assert.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrConcurrentModification)
}

func TestMemoryRepository_FindByStatus(t *testing.T) {
	// Setup
	repo := NewMemoryRepository()
	ctx := context.Background()

	// Create and save multiple ads with different statuses
	for i := 1; i <= 5; i++ {
		ad := &models.Ad{
			AdID: "status-test-ad-" + string(rune(i+'0')),
			AdSubmission: models.AdSubmission{
				Title:       "Test Ad " + string(rune(i+'0')),
				Description: "Test description",
			},
			Status:    models.StatusSubmitted,
			CreatedAt: time.Now().UTC(),
			Version:   1,
		}

		// Alternate between submitted and queued
		if i%2 == 0 {
			ad.Status = models.StatusQueued
		}

		_, err := repo.Save(ctx, ad)
		assert.NoError(t, err)
	}

	// Test
	submittedAds, err := repo.FindByStatus(ctx, models.StatusSubmitted, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, submittedAds, 3) // 3 submitted ads (1, 3, 5)

	queuedAds, err := repo.FindByStatus(ctx, models.StatusQueued, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, queuedAds, 2) // 2 queued ads (2, 4)
}

func TestMemoryRepository_AcquireLock(t *testing.T) {
	// Setup
	repo := NewMemoryRepository()
	ctx := context.Background()
	adID := "lock-test-ad"
	ownerID1 := "worker-1"
	ownerID2 := "worker-2"

	// Test 1: First lock should succeed
	acquired, err := repo.AcquireLock(ctx, adID, ownerID1, 30)
	assert.NoError(t, err)
	assert.True(t, acquired)

	// Test 2: Second lock from different owner should fail
	acquired, err = repo.AcquireLock(ctx, adID, ownerID2, 30)
	assert.NoError(t, err)
	assert.False(t, acquired)

	// Test 3: Same owner should be able to refresh the lock
	acquired, err = repo.AcquireLock(ctx, adID, ownerID1, 30)
	assert.NoError(t, err)
	assert.True(t, acquired)

	// Test 4: Release the lock and verify another owner can acquire it
	released, err := repo.ReleaseLock(ctx, adID, ownerID1)
	assert.NoError(t, err)
	assert.True(t, released)

	acquired, err = repo.AcquireLock(ctx, adID, ownerID2, 30)
	assert.NoError(t, err)
	assert.True(t, acquired)
}
