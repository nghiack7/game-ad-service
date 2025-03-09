package dtos

import (
	"net/http"

	"github.com/nghiack7/game-ad-service/pkg/utils/response"
)

type Base interface {
	JSON(w http.ResponseWriter, data interface{})
	HandleError(w http.ResponseWriter, err error)
}

type base struct{}

var _ Base = (*base)(nil)

func NewBase() Base {
	return &base{}
}

func (b *base) JSON(w http.ResponseWriter, data interface{}) {
	response.JSON(w, data)
}

func (b *base) HandleError(w http.ResponseWriter, err error) {
	response.HandleError(w, err)
}
