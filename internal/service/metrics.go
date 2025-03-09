package service

// MetricsService is a service that provides metrics for the application.
type MetricsService interface {
	IncAdSubmit(status string)
	IncAdGet(status string)
	IncAdList(status string)
	IncAdListByStatus(status string)
}
