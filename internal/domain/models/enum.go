package models

type AdStatus string
type Priority int

const (
	// StatusSubmitted indicates the ad has been received but not yet queued for processing
	StatusSubmitted AdStatus = "submitted"

	// StatusQueued indicates the ad is in the processing queue awaiting analysis
	StatusQueued AdStatus = "queued"

	// StatusProcessing indicates the ad is currently being analyzed
	StatusProcessing AdStatus = "processing"

	// StatusCompleted indicates the ad has been successfully processed
	StatusCompleted AdStatus = "completed"

	// StatusFailed indicates the ad processing has failed
	StatusFailed AdStatus = "failed"
)

const (
	PriorityHighest Priority = iota + 1
	PriorityHigh
	PriorityMedium
	PriorityLow
	PriorityLowest
)
