package errors

import (
	"fmt"
	"net/http"
)

// Module constants definition.
const (
	Moduleshared = 00
	AdsModule    = 01
)

// shared module error codes definition.
var (
	ErrInvalidRequest = fmtErrorCode(http.StatusBadRequest, Moduleshared, 1)
	ErrUnauthorized   = fmtErrorCode(http.StatusUnauthorized, Moduleshared, 1)
	ErrInternalServer = fmtErrorCode(http.StatusInternalServerError, Moduleshared, 1)
	ErrNoResponse     = fmtErrorCode(http.StatusInternalServerError, Moduleshared, 2)
	ErrNotFound       = fmtErrorCode(http.StatusNotFound, Moduleshared, 1)
	ErrConflict       = fmtErrorCode(http.StatusConflict, Moduleshared, 1)
)
var (
	ErrQueueFull = fmtErrorCode(http.StatusTooManyRequests, Moduleshared, 1)
)

// GetErrorMessage gets error message from errorMessageMap.
func GetErrorMessage(errCode ErrorCode, args ...interface{}) string {
	if len(args) > 0 {
		msg, ok := args[0].(string)
		if ok {
			return msg
		}
	}

	msg := http.StatusText(errCode.Status())
	return fmt.Sprintf(msg, args...)
}
