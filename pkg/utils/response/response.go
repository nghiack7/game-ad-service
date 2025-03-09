package response

import (
	"encoding/json"
	"net/http"

	"github.com/nghiack7/game-ad-service/pkg/errors"
)

// JSON responds a HTTP request with JSON data.
func JSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if data != nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(data)
	} else {
		w.WriteHeader(errors.ErrNoResponse.Status())
		json.NewEncoder(w).Encode(errors.New(errors.ErrNoResponse))
	}
}

// HandleError handles error of HTTP request.
func HandleError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		appErr, ok := err.(errors.AppError)
		if ok {
			w.WriteHeader(appErr.ErrorCode.Status())
			json.NewEncoder(w).Encode(appErr)
		} else {
			w.WriteHeader(errors.ErrInternalServer.Status())
			json.NewEncoder(w).Encode(errors.New(errors.ErrInternalServer))
		}
	} else {
		w.WriteHeader(errors.ErrNoResponse.Status())
		json.NewEncoder(w).Encode(errors.New(errors.ErrNoResponse))
	}
}

// HandleErrorWithoutContext return error response without context
func HandleErrorWithoutContext(err error) string {
	appErr, ok := err.(errors.AppError)
	if ok {
		data, _ := json.Marshal(appErr)
		return string(data)
	}

	return errors.New(errors.ErrInternalServer).Error()
}
