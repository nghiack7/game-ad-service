package models

import (
	"time"
)

type AdSubmission struct {
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

	// Priority level (lower number = higher priority, 1 is highest)
	Priority int `json:"priority,omitempty"`
}

// Ad represents a processed or in-process advertisement
type Ad struct {
	// Unique identifier for the ad
	AdID string `json:"adId"`

	// The original submission data
	AdSubmission

	// Current status of the ad in the processing pipeline
	Status AdStatus `json:"status"`

	// When the ad was created
	CreatedAt time.Time `json:"createdAt"`

	// When the ad was queued for processing
	QueuedAt *time.Time `json:"queuedAt,omitempty"`

	// When the ad started processing
	ProcessingStartedAt *time.Time `json:"processingStartedAt,omitempty"`

	// When the ad processing was completed (if applicable)
	CompletedAt *time.Time `json:"completedAt,omitempty"`

	// Analysis results (available after processing)
	Analysis *AdAnalysis `json:"analysis,omitempty"`

	// Error message if processing failed
	Error string `json:"error,omitempty"`

	// Number of processing attempts
	Attempts int `json:"-"`

	// Worker ID processing this ad (if any)
	WorkerID string `json:"-"`

	// For optimistic concurrency control
	Version int64 `json:"-"`

	// Effective priority (adjusted for aging)
	EffectivePriority int `json:"-"`
}

// AdAnalysis contains the results of the ad effectiveness analysis
type AdAnalysis struct {
	// Overall effectiveness score (0-10)
	EffectivenessScore float64 `json:"effectivenessScore"`

	// Identified strengths of the advertisement
	Strengths []string `json:"strengths"`

	// Suggestions for improving the advertisement
	ImprovementSuggestions []string `json:"improvementSuggestions"`
}

// NewAd creates a new Ad from a submission
func NewAd(submission AdSubmission) *Ad {
	// Set default priority if not provided
	if submission.Priority <= 0 {
		submission.Priority = 5 // Default priority (middle)
	}

	now := time.Now().UTC()

	return &Ad{
		AdSubmission: submission,
		Status:       StatusSubmitted,
		CreatedAt:    now,
		Attempts:     0,
		Version:      1,
	}
}

// SetStatus updates the ad status and sets appropriate timestamps
func (a *Ad) SetStatus(newStatus AdStatus) {
	a.Status = newStatus

	// Set timestamps based on status transitions
	now := time.Now().UTC()

	if newStatus == StatusQueued {
		a.QueuedAt = &now
		// Reset effective priority to original priority when queued
		a.EffectivePriority = a.Priority
	} else if newStatus == StatusProcessing {
		a.ProcessingStartedAt = &now
	} else if newStatus == StatusCompleted {
		a.CompletedAt = &now
	}

}

// IncrementAttempts increases the processing attempt counter
func (a *Ad) IncrementAttempts() int {
	a.Attempts++
	return a.Attempts
}

// AssignWorker assigns this ad to a specific worker
func (a *Ad) AssignWorker(workerID string) {
	a.WorkerID = workerID
}

// SetAnalysisResults updates the ad with analysis results
func (a *Ad) SetAnalysisResults(analysis AdAnalysis) {
	a.Analysis = &analysis
}

// SetError sets an error message and marks the ad as failed
func (a *Ad) SetError(errorMsg string) {
	a.Error = errorMsg
	a.SetStatus(StatusFailed)
}

// IsProcessable checks if this ad can be processed
func (a *Ad) IsProcessable() bool {
	return a.Status == StatusSubmitted || a.Status == StatusQueued
}

// CanBeRequeued determines if a failed ad can be requeued based on attempt count
func (a *Ad) CanBeRequeued(maxAttempts int) bool {
	return a.Status == StatusFailed && a.Attempts < maxAttempts
}

// CalculateEffectivePriority calculates the effective priority based on waiting time
// Lower numbers = higher priority
func (a *Ad) CalculateEffectivePriority(agingRate float64, maxWaitTime time.Duration) int {
	// If not queued or no QueuedAt timestamp, return original priority
	if a.Status != StatusQueued || a.QueuedAt == nil {
		return a.Priority
	}

	// Calculate how long the ad has been waiting
	waitingTime := time.Since(*a.QueuedAt)

	// Calculate priority boost based on waiting time
	// The longer it waits, the more its priority is boosted (lower number = higher priority)
	waitingRatio := float64(waitingTime) / float64(maxWaitTime)
	if waitingRatio > 1.0 {
		waitingRatio = 1.0 // Cap at 1.0 (max boost)
	}

	// Calculate priority boost (higher agingRate = faster aging)
	// For very long waits, this can boost even low priority items to high priority
	// The formula is designed to give a larger boost to higher priority numbers (lower priority)
	priorityBoost := int(float64(a.Priority) * waitingRatio * agingRate)

	// Calculate effective priority (original priority minus boost)
	// This ensures that lower numbers (higher priority) remain lower
	// and higher numbers (lower priority) get reduced more aggressively
	effectivePriority := a.Priority - priorityBoost
	if effectivePriority < 1 {
		effectivePriority = 1 // Minimum priority is 1 (highest)
	}

	a.EffectivePriority = effectivePriority
	return effectivePriority
}

// GetWaitingTime returns how long the ad has been waiting in the queue
func (a *Ad) GetWaitingTime() time.Duration {
	if a.QueuedAt == nil {
		return 0
	}
	return time.Since(*a.QueuedAt)
}

// HasExceededMaxWaitTime checks if the ad has been waiting longer than the maximum wait time
func (a *Ad) HasExceededMaxWaitTime(maxWaitTime time.Duration) bool {
	return a.GetWaitingTime() > maxWaitTime
}
