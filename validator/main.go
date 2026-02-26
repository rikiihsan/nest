package validator

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/gofiber/fiber/v2"
)

var (
	ens      = en.New()
	uni      = ut.New(ens, ens)
	trans, _ = uni.GetTranslator("en")
	validate = validator.New()
)

// Initialize validator on package load
func init() {
	// Register tag name function for better JSON field mapping
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return fld.Name
		}
		return name
	})

	// Register default translations
	en_translations.RegisterDefaultTranslations(validate, trans)
}

// GetFieldTag extracts field name from struct tag
func GetFieldTag(data interface{}, fieldName string, sourceTag string) string {
	t := reflect.TypeOf(data)
	if t == nil {
		return fieldName
	}

	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Only process struct types
	if t.Kind() != reflect.Struct {
		return fieldName
	}

	field, found := t.FieldByName(fieldName)
	if !found {
		return fieldName
	}

	tagValue := field.Tag.Get(sourceTag)
	if tagValue == "" || tagValue == "-" {
		// Fallback to json tag if source tag not found
		if sourceTag != "json" {
			jsonTag := field.Tag.Get("json")
			if jsonTag != "" && jsonTag != "-" {
				return strings.Split(jsonTag, ",")[0]
			}
		}
		return fieldName
	}

	// Return first part of tag (before comma)
	return strings.Split(tagValue, ",")[0]
}

// Init initializes validator with custom translators
func Init(translators ...Translator) error {
	// Re-register default translations
	if err := en_translations.RegisterDefaultTranslations(validate, trans); err != nil {
		return fmt.Errorf("failed to register default translations: %w", err)
	}

	// Register custom translations
	for _, item := range translators {
		err := validate.RegisterTranslation(item.Tag, trans,
			func(ut ut.Translator) error {
				return ut.Add(item.Tag, item.Message, true)
			},
			func(ut ut.Translator, fe validator.FieldError) string {
				t, _ := ut.T(item.Tag, fe.Field())
				return t
			})
		if err != nil {
			return fmt.Errorf("failed to register translation for tag %s: %w", item.Tag, err)
		}
	}

	return nil
}

// Validate validates a struct and returns validation errors
func Validate(data interface{}, source string) []ValidatorError {
	if data == nil {
		return []ValidatorError{}
	}

	validationErrors := []ValidatorError{}
	errs := validate.Struct(data)

	if errs != nil {
		// Type assertion with safety check
		if validationErrs, ok := errs.(validator.ValidationErrors); ok {
			for _, err := range validationErrs {
				elem := ValidatorError{
					FailedField: GetFieldTag(data, err.Field(), source),
					Tag:         err.Tag(),
					Message:     err.Translate(trans),
				}
				validationErrors = append(validationErrors, elem)
			}
		}
	}

	return validationErrors
}

// ValidateWithContext validates a struct with context for timeout handling
func ValidateWithContext(ctx context.Context, data interface{}, source string, timeout time.Duration) []ValidatorError {
	if data == nil {
		return []ValidatorError{}
	}

	// Create context with timeout if specified
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// Channel to receive validation result
	resultChan := make(chan []ValidatorError, 1)

	// Run validation in goroutine
	go func() {
		result := Validate(data, source)
		select {
		case resultChan <- result:
		case <-ctx.Done():
		}
	}()

	// Wait for result or context cancellation
	select {
	case result := <-resultChan:
		return result
	case <-ctx.Done():
		return []ValidatorError{{
			FailedField: "validation",
			Tag:         "timeout",
			Message:     "Validation timeout exceeded",
		}}
	}
}

// SliceValidate validates a slice of structs
func SliceValidate(data interface{}, source string) []ValidatorError {
	if data == nil {
		return []ValidatorError{}
	}

	validationErrors := []ValidatorError{}
	v := reflect.ValueOf(data)

	// Check if data is slice or array
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return []ValidatorError{{
			FailedField: "root",
			Tag:         "slice",
			Message:     "Expected slice or array",
		}}
	}

	// Validate each element in slice
	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)

		// Handle interface{} and pointer types
		if elem.Kind() == reflect.Interface {
			elem = elem.Elem()
		}

		if !elem.IsValid() {
			continue
		}

		elemData := elem.Interface()
		errs := validate.Struct(elemData)

		if errs != nil {
			if validationErrs, ok := errs.(validator.ValidationErrors); ok {
				for _, err := range validationErrs {
					validationError := ValidatorError{
						FailedField: fmt.Sprintf("[%d].%s", i, GetFieldTag(elemData, err.Field(), source)),
						Tag:         err.Tag(),
						Message:     fmt.Sprintf("Index %d: %s", i, err.Translate(trans)),
						Index:       &i,
					}
					validationErrors = append(validationErrors, validationError)
				}
			}
		}
	}

	return validationErrors
}

// ValidateVar validates a single variable with validation tags
func ValidateVar(field interface{}, tag string, fieldName string) []ValidatorError {
	validationErrors := []ValidatorError{}
	err := validate.Var(field, tag)

	if err != nil {
		if validationErrs, ok := err.(validator.ValidationErrors); ok {
			for _, verr := range validationErrs {
				elem := ValidatorError{
					FailedField: fieldName,
					Tag:         verr.Tag(),
					Message:     verr.Translate(trans),
				}
				validationErrors = append(validationErrors, elem)
			}
		}
	}

	return validationErrors
}

// AddCustomValidation adds custom validation rule
func AddCustomValidation(tag string, fn validator.Func, message string) error {
	// Register validation function
	err := validate.RegisterValidation(tag, fn)
	if err != nil {
		return fmt.Errorf("failed to register validation function: %w", err)
	}

	// Register translation
	err = validate.RegisterTranslation(tag, trans,
		func(ut ut.Translator) error {
			return ut.Add(tag, message, true)
		},
		func(ut ut.Translator, fe validator.FieldError) string {
			t, _ := ut.T(tag, fe.Field())
			return t
		})
	if err != nil {
		return fmt.Errorf("failed to register validation translation: %w", err)
	}

	return nil
}

// GetValidator returns the underlying validator instance for advanced usage
func GetValidator() *validator.Validate {
	return validate
}

// GetTranslator returns the current translator instance
func GetTranslator() ut.Translator {
	return trans
}

// NewValidators creates a new Validators instance
func NewValidators(ctx *fiber.Ctx, data interface{}) *Validators {
	return &Validators{
		Ctx:            ctx,
		Data:           data,
		Error:          false,
		ValidationsErr: []ValidatorError{},
	}
}

// ValidateStruct is a convenience method for Validators struct
func (v *Validators) ValidateStruct(source string) {
	errors := Validate(v.Data, source)
	v.ValidationsErr = append(v.ValidationsErr, errors...)
	if len(errors) > 0 {
		v.Error = true
	}
}

// ValidateSlice is a convenience method for Validators struct
func (v *Validators) ValidateSlice(source string) {
	errors := SliceValidate(v.Data, source)
	v.ValidationsErr = append(v.ValidationsErr, errors...)
	if len(errors) > 0 {
		v.Error = true
	}
}
