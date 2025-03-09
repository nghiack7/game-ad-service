package dtos

import (
	"time"

	"github.com/nghiack7/game-ad-service/internal/domain/models"
)

// AdSubmissionDTO represents the data needed to submit a new advertisement for analysis
type AdSubmissionDTO struct {
	// Title of the game advertisement
	Title string `json:"title" validate:"required"`

	// Description of the game
	Description string `json:"description" validate:"required"`

	// Genre of the game
	Genre string `json:"genre"`

	// TargetAudience describes the intended audience segments
	TargetAudience []string `json:"targetAudience"`

	// VisualElements describes key visual components in the ad
	VisualElements []string `json:"visualElements"`

	// CallToAction is the action prompt in the advertisement
	CallToAction string `json:"callToAction"`

	// Duration in seconds of the advertisement
	Duration int `json:"duration"`

	// Priority level (lower number = higher priority, 1 is highest)
	Priority int `json:"priority,omitempty"`
}

// AdAnalysisDTO contains the results of the ad effectiveness analysis
type AdAnalysisDTO struct {
	// Overall effectiveness score (0-10)
	EffectivenessScore float64 `json:"effectivenessScore"`

	// Identified strengths of the advertisement
	Strengths []string `json:"strengths"`

	// Suggestions for improving the advertisement
	ImprovementSuggestions []string `json:"improvementSuggestions"`
}

// AdDTO represents a processed or in-process advertisement for API responses
type AdDTO struct {
	// Unique identifier for the ad
	AdID string `json:"adId"`

	// Title of the game advertisement
	Title string `json:"title"`

	// Description of the game
	Description string `json:"description"`

	// Genre of the game
	Genre string `json:"genre"`

	// TargetAudience describes the intended audience segments
	TargetAudience []string `json:"targetAudience"`

	// VisualElements describes key visual components in the ad
	VisualElements []string `json:"visualElements"`

	// CallToAction is the action prompt in the advertisement
	CallToAction string `json:"callToAction"`

	// Duration in seconds of the advertisement
	Duration int `json:"duration"`

	// Priority level
	Priority int `json:"priority,omitempty"`

	// Current status of the ad in the processing pipeline
	Status models.AdStatus `json:"status"`

	// When the ad was created
	CreatedAt time.Time `json:"createdAt"`

	// When the ad processing was completed (if applicable)
	CompletedAt *time.Time `json:"completedAt,omitempty"`

	// Analysis results (available after processing)
	Analysis *AdAnalysisDTO `json:"analysis,omitempty"`

	// Error message if processing failed
	Error string `json:"error,omitempty"`
}

// AdListItemDTO represents a simplified ad for list views
type AdListItemDTO struct {
	// Unique identifier for the ad
	AdID string `json:"adId"`

	// Title of the game advertisement
	Title string `json:"title"`

	// Genre of the game
	Genre string `json:"genre"`

	// Current status of the ad in the processing pipeline
	Status models.AdStatus `json:"status"`

	// When the ad was created
	CreatedAt time.Time `json:"createdAt"`

	// When the ad processing was completed (if applicable)
	CompletedAt *time.Time `json:"completedAt,omitempty"`

	// Overall effectiveness score (0-10), if analysis is complete
	EffectivenessScore *float64 `json:"effectivenessScore,omitempty"`
}

// AdStatusUpdateDTO represents a status update for an ad
type AdStatusUpdateDTO struct {
	// New status to set
	Status models.AdStatus `json:"status"`
}

// AdErrorDTO represents an error response for ad processing
type AdErrorDTO struct {
	// Error message
	Message string `json:"message"`
}

// AdStatsDTO represents statistics about ads in the system
type AdStatsDTO struct {
	// Total number of ads in the system
	TotalAds int `json:"totalAds"`

	// Count of ads by status
	StatusCounts map[models.AdStatus]int `json:"statusCounts"`

	// Average processing time in seconds
	AvgProcessingTime float64 `json:"avgProcessingTime,omitempty"`

	// Average effectiveness score of completed ads
	AvgEffectivenessScore float64 `json:"avgEffectivenessScore,omitempty"`
}
