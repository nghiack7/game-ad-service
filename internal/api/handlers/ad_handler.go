package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/nghiack7/game-ad-service/internal/domain/models"
	"github.com/nghiack7/game-ad-service/internal/interfaces/dtos"
	"github.com/nghiack7/game-ad-service/internal/interfaces/mapper"
	"github.com/nghiack7/game-ad-service/internal/service"
	"github.com/nghiack7/game-ad-service/pkg/errors"
	"github.com/nghiack7/game-ad-service/pkg/utils"
)

// AdHandler handles ad-related HTTP requests
type AdHandler struct {
	dtos.Base
	adService service.AdService
}

// NewAdHandler creates a new ad handler
func NewAdHandler(adService service.AdService) *AdHandler {
	return &AdHandler{
		Base:      dtos.NewBase(),
		adService: adService,
	}
}

// RegisterRoutes registers the ad handler routes with the router
func (h *AdHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/ads/stats", h.getStats).Methods(http.MethodGet)

	router.HandleFunc("/ads", h.submitAd).Methods(http.MethodPost)
	router.HandleFunc("/ads/{id}", h.getAd).Methods(http.MethodGet)

	// Optional extended endpoints
	router.HandleFunc("/ads", h.listAds).Methods(http.MethodGet)
	router.HandleFunc("/ads/status/{status}", h.listAdsByStatus).Methods(http.MethodGet)
}

// SubmitAd handles a new ad submission
func (h *AdHandler) submitAd(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var submission dtos.AdSubmissionDTO
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&submission); err != nil {
		h.HandleError(w, err)
		return
	}
	defer r.Body.Close()

	// Basic validation
	if err := utils.Validate(submission); err != nil {
		h.HandleError(w, err)
		return
	}

	// Create new ad
	adSubmission := mapper.MapAdSubmissionToAdSubmission(submission)
	ad := models.NewAd(adSubmission)
	ad.AdID = utils.GenerateID()
	ad.CreatedAt = time.Now().UTC()

	// Submit to service
	res, err := h.adService.SubmitAd(r.Context(), ad)
	if err != nil {
		h.HandleError(w, err)
		return
	}

	h.JSON(w, res)
}

// GetAd retrieves the status and results for a specific ad
func (h *AdHandler) getAd(w http.ResponseWriter, r *http.Request) {
	// Get ad ID from URL
	vars := mux.Vars(r)
	adID := vars["id"]

	if adID == "" {
		h.HandleError(w, errors.New(errors.ErrInvalidRequest))
		return
	}

	// Retrieve ad from service
	ad, err := h.adService.GetAd(r.Context(), adID)
	if err != nil {
		h.HandleError(w, err)
		return
	}

	h.JSON(w, ad)
}

// ListAds lists all ads with optional filtering
func (h *AdHandler) listAds(w http.ResponseWriter, r *http.Request) {
	// Get query parameters for pagination
	limit := getIntQueryParam(r, "limit", 10)
	offset := getIntQueryParam(r, "offset", 0)

	// Get status filter if provided
	status := r.URL.Query().Get("status")

	// Get ads from service
	ads, err := h.adService.ListAds(r.Context(), status, limit, offset)
	if err != nil {
		h.HandleError(w, err)
		return
	}

	// Create response with metadata
	res := map[string]interface{}{
		"ads":    ads,
		"limit":  limit,
		"offset": offset,
		"count":  len(ads),
	}

	h.JSON(w, res)
}

// ListAdsByStatus lists ads with a specific status
func (h *AdHandler) listAdsByStatus(w http.ResponseWriter, r *http.Request) {
	// Get status from URL
	vars := mux.Vars(r)
	status := vars["status"]

	// Get query parameters for pagination
	limit := getIntQueryParam(r, "limit", 10)
	offset := getIntQueryParam(r, "offset", 0)

	// Get ads from service
	ads, err := h.adService.ListAds(r.Context(), status, limit, offset)
	if err != nil {
		h.HandleError(w, err)
		return
	}

	// Create response with metadata
	res := map[string]interface{}{
		"status": status,
		"ads":    ads,
		"limit":  limit,
		"offset": offset,
		"count":  len(ads),
	}

	h.JSON(w, res)
}

// Helper functions

// getIntQueryParam gets an integer query parameter with a default value
func getIntQueryParam(r *http.Request, name string, defaultValue int) int {
	values := r.URL.Query()[name]
	if len(values) == 0 {
		return defaultValue
	}

	var value int
	_, err := fmt.Sscanf(values[0], "%d", &value)
	if err != nil || value < 0 {
		return defaultValue
	}

	return value
}

// GetStats returns processing statistics
func (h *AdHandler) getStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.adService.GetStats(r.Context())
	if err != nil {
		h.HandleError(w, err)
		return
	}

	h.JSON(w, stats)
}
