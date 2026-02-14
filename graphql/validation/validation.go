package validation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/JIeeiroSst/utils/graphql/errors"
)

type Validator interface {
	Validate() error
}

type FieldValidator struct {
	Field   string
	Value   interface{}
	Errors  []string
	isValid bool
}

func NewFieldValidator(field string, value interface{}) *FieldValidator {
	return &FieldValidator{
		Field:   field,
		Value:   value,
		Errors:  []string{},
		isValid: true,
	}
}

func (fv *FieldValidator) addError(message string) *FieldValidator {
	fv.Errors = append(fv.Errors, message)
	fv.isValid = false
	return fv
}

func (fv *FieldValidator) Required() *FieldValidator {
	if fv.Value == nil {
		return fv.addError(fmt.Sprintf("%s is required", fv.Field))
	}

	if str, ok := fv.Value.(string); ok && strings.TrimSpace(str) == "" {
		return fv.addError(fmt.Sprintf("%s cannot be empty", fv.Field))
	}

	return fv
}

func (fv *FieldValidator) MinLength(min int) *FieldValidator {
	if str, ok := fv.Value.(string); ok {
		if len(str) < min {
			return fv.addError(fmt.Sprintf("%s must be at least %d characters", fv.Field, min))
		}
	}
	return fv
}

func (fv *FieldValidator) MaxLength(max int) *FieldValidator {
	if str, ok := fv.Value.(string); ok {
		if len(str) > max {
			return fv.addError(fmt.Sprintf("%s must not exceed %d characters", fv.Field, max))
		}
	}
	return fv
}

func (fv *FieldValidator) Email() *FieldValidator {
	if str, ok := fv.Value.(string); ok {
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(str) {
			return fv.addError(fmt.Sprintf("%s must be a valid email address", fv.Field))
		}
	}
	return fv
}

func (fv *FieldValidator) URL() *FieldValidator {
	if str, ok := fv.Value.(string); ok {
		urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
		if !urlRegex.MatchString(str) {
			return fv.addError(fmt.Sprintf("%s must be a valid URL", fv.Field))
		}
	}
	return fv
}

func (fv *FieldValidator) Min(min int) *FieldValidator {
	switch v := fv.Value.(type) {
	case int:
		if v < min {
			return fv.addError(fmt.Sprintf("%s must be at least %d", fv.Field, min))
		}
	case int32:
		if int(v) < min {
			return fv.addError(fmt.Sprintf("%s must be at least %d", fv.Field, min))
		}
	case int64:
		if int(v) < min {
			return fv.addError(fmt.Sprintf("%s must be at least %d", fv.Field, min))
		}
	}
	return fv
}

func (fv *FieldValidator) Max(max int) *FieldValidator {
	switch v := fv.Value.(type) {
	case int:
		if v > max {
			return fv.addError(fmt.Sprintf("%s must not exceed %d", fv.Field, max))
		}
	case int32:
		if int(v) > max {
			return fv.addError(fmt.Sprintf("%s must not exceed %d", fv.Field, max))
		}
	case int64:
		if int(v) > max {
			return fv.addError(fmt.Sprintf("%s must not exceed %d", fv.Field, max))
		}
	}
	return fv
}

func (fv *FieldValidator) Pattern(pattern string, message string) *FieldValidator {
	if str, ok := fv.Value.(string); ok {
		matched, err := regexp.MatchString(pattern, str)
		if err != nil || !matched {
			if message == "" {
				message = fmt.Sprintf("%s has invalid format", fv.Field)
			}
			return fv.addError(message)
		}
	}
	return fv
}

func (fv *FieldValidator) OneOf(allowed []string) *FieldValidator {
	if str, ok := fv.Value.(string); ok {
		for _, a := range allowed {
			if str == a {
				return fv
			}
		}
		return fv.addError(fmt.Sprintf("%s must be one of: %s", fv.Field, strings.Join(allowed, ", ")))
	}
	return fv
}

func (fv *FieldValidator) Custom(fn func(interface{}) error) *FieldValidator {
	if err := fn(fv.Value); err != nil {
		return fv.addError(err.Error())
	}
	return fv
}

func (fv *FieldValidator) IsValid() bool {
	return fv.isValid
}

type ValidationResult struct {
	fields map[string][]string
	valid  bool
}

func NewValidationResult() *ValidationResult {
	return &ValidationResult{
		fields: make(map[string][]string),
		valid:  true,
	}
}

func (vr *ValidationResult) AddField(fv *FieldValidator) *ValidationResult {
	if !fv.IsValid() {
		vr.fields[fv.Field] = fv.Errors
		vr.valid = false
	}
	return vr
}

func (vr *ValidationResult) AddError(field, message string) *ValidationResult {
	if vr.fields[field] == nil {
		vr.fields[field] = []string{}
	}
	vr.fields[field] = append(vr.fields[field], message)
	vr.valid = false
	return vr
}

func (vr *ValidationResult) IsValid() bool {
	return vr.valid
}

func (vr *ValidationResult) Error() error {
	if vr.valid {
		return nil
	}

	fieldErrors := make(map[string]string)
	for field, errs := range vr.fields {
		fieldErrors[field] = strings.Join(errs, "; ")
	}

	return errors.ValidationErrors(fieldErrors)
}

func ValidateEmail(email string) error {
	fv := NewFieldValidator("email", email).Email()
	if !fv.IsValid() {
		return fmt.Errorf(strings.Join(fv.Errors, "; "))
	}
	return nil
}

func ValidateRequired(field string, value interface{}) error {
	fv := NewFieldValidator(field, value).Required()
	if !fv.IsValid() {
		return fmt.Errorf(strings.Join(fv.Errors, "; "))
	}
	return nil
}

func ValidateStringLength(field string, value string, min, max int) error {
	fv := NewFieldValidator(field, value).MinLength(min).MaxLength(max)
	if !fv.IsValid() {
		return fmt.Errorf(strings.Join(fv.Errors, "; "))
	}
	return nil
}
