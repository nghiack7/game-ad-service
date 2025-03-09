package utils

import (
	"github.com/go-playground/validator/v10"
)

// Validator defines an interface for validation operations
type Validator interface {
	// ValidateStruct validates a struct against validation tags
	ValidateStruct(v interface{}) error

	// RegisterCustomValidation adds a custom validation function
	RegisterCustomValidation(tag string, fn validator.Func) error
}

// defaultValidator implements the Validator interface
type defaultValidator struct {
	validate *validator.Validate
}

// NewValidator creates a new instance of the validator
func NewValidator() Validator {
	return &defaultValidator{
		validate: validator.New(),
	}
}

// ValidateStruct validates a struct against validation tags
func (v *defaultValidator) ValidateStruct(s interface{}) error {
	return v.validate.Struct(s)
}

// RegisterCustomValidation adds a custom validation function
func (v *defaultValidator) RegisterCustomValidation(tag string, fn validator.Func) error {
	return v.validate.RegisterValidation(tag, fn)
}

// For backward compatibility
var defaultInstance = NewValidator()

func Validate(v interface{}) error {
	return defaultInstance.ValidateStruct(v)
}
