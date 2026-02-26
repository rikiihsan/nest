package validator

import (
	"github.com/gofiber/fiber/v2"
)

// Translator represents custom validation message translator
type Translator struct {
	Tag     string
	Message string
}

// ValidatorError represents validation error structure
type ValidatorError struct {
	FailedField string `json:"failed_field"`
	Tag         string `json:"tag"`
	Message     string `json:"message"`
	Index       *int   `json:"index,omitempty"` // For slice validation
}

// Validators holds validation context and results
type Validators struct {
	Ctx            *fiber.Ctx
	Data           interface{}
	Error          bool
	ValidationsErr []ValidatorError
}

// HasErrors checks if there are validation errors
func (v *Validators) HasErrors() bool {
	return len(v.ValidationsErr) > 0
}

// GetFirstError returns the first validation error
func (v *Validators) GetFirstError() *ValidatorError {
	if len(v.ValidationsErr) > 0 {
		return &v.ValidationsErr[0]
	}
	return nil
}

// AddError adds custom validation error
func (v *Validators) AddError(field, tag, message string) {
	v.ValidationsErr = append(v.ValidationsErr, ValidatorError{
		FailedField: field,
		Tag:         tag,
		Message:     message,
	})
	v.Error = true
}

// AddErrorWithIndex adds custom validation error with index for slice validation
func (v *Validators) AddErrorWithIndex(field, tag, message string, index int) {
	v.ValidationsErr = append(v.ValidationsErr, ValidatorError{
		FailedField: field,
		Tag:         tag,
		Message:     message,
		Index:       &index,
	})
	v.Error = true
}

// GetErrorsByField returns errors for specific field
func (v *Validators) GetErrorsByField(field string) []ValidatorError {
	var errors []ValidatorError
	for _, err := range v.ValidationsErr {
		if err.FailedField == field {
			errors = append(errors, err)
		}
	}
	return errors
}

// Clear clears all validation errors
func (v *Validators) Clear() {
	v.ValidationsErr = []ValidatorError{}
	v.Error = false
}
