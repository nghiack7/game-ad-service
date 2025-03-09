package api

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/nghiack7/game-ad-service/internal/api/handlers"
	"github.com/nghiack7/game-ad-service/internal/api/middleware"
	"github.com/nghiack7/game-ad-service/internal/service"
	"github.com/nghiack7/game-ad-service/pkg/logger"
)

// Router handles HTTP routing
type Router struct {
	router *mux.Router
	logger logger.Logger
}

// ServeHTTP implements the http.Handler interface
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.router.ServeHTTP(w, req)
}

// NewRouter creates a new HTTP router
func NewRouter(adService service.AdService, logger logger.Logger) *Router {
	router := mux.NewRouter()

	// Add middleware
	router.Use(middleware.LoggingMiddleware(logger))
	router.Use(middleware.RecoveryMiddleware(logger))
	router.Use(middleware.CORSMiddleware())

	// Create handlers
	adHandler := handlers.NewAdHandler(adService)
	// Register routes
	adHandler.RegisterRoutes(router)
	router.HandleFunc("/health", healthCheckHandler).Methods(http.MethodGet)
	router.NotFoundHandler = http.HandlerFunc(notFoundHandler)
	return &Router{
		router: router,
		logger: logger,
	}
}

// healthCheckHandler handles health check requests
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok","timestamp":"` + time.Now().UTC().Format(time.RFC3339) + `"}`))
}

// notFoundHandler handles 404 requests
func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"success":false,"error":"Not Found","path":"` + r.URL.Path + `"}`))
}
