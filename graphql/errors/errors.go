package errors

import (
	"fmt"
)

type ErrorCode string

const (
	ErrorCodeInternal     ErrorCode = "INTERNAL_ERROR"
	ErrorCodeNotFound     ErrorCode = "NOT_FOUND"
	ErrorCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	ErrorCodeForbidden    ErrorCode = "FORBIDDEN"
	ErrorCodeBadRequest   ErrorCode = "BAD_REQUEST"
	ErrorCodeValidation   ErrorCode = "VALIDATION_ERROR"
	ErrorCodeConflict     ErrorCode = "CONFLICT"
	ErrorCodeRateLimit    ErrorCode = "RATE_LIMIT_EXCEEDED"
)

type GraphQLError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

func (e *GraphQLError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func NewError(code ErrorCode, message string) *GraphQLError {
	return &GraphQLError{
		Code:       code,
		Message:    message,
		Extensions: make(map[string]interface{}),
	}
}

func (e *GraphQLError) WithExtension(key string, value interface{}) *GraphQLError {
	if e.Extensions == nil {
		e.Extensions = make(map[string]interface{})
	}
	e.Extensions[key] = value
	return e
}

func NotFound(resource string) *GraphQLError {
	return NewError(ErrorCodeNotFound, fmt.Sprintf("%s not found", resource))
}

func NotFoundWithID(resource, id string) *GraphQLError {
	err := NewError(ErrorCodeNotFound, fmt.Sprintf("%s with ID %s not found", resource, id))
	return err.WithExtension("resource", resource).WithExtension("id", id)
}

func Unauthorized(message string) *GraphQLError {
	if message == "" {
		message = "Authentication required"
	}
	return NewError(ErrorCodeUnauthorized, message)
}

func Forbidden(message string) *GraphQLError {
	if message == "" {
		message = "Access forbidden"
	}
	return NewError(ErrorCodeForbidden, message)
}

func BadRequest(message string) *GraphQLError {
	return NewError(ErrorCodeBadRequest, message)
}

func ValidationError(field, message string) *GraphQLError {
	err := NewError(ErrorCodeValidation, message)
	return err.WithExtension("field", field)
}

func ValidationErrors(errors map[string]string) *GraphQLError {
	err := NewError(ErrorCodeValidation, "Validation failed")
	return err.WithExtension("fields", errors)
}

func Conflict(message string) *GraphQLError {
	return NewError(ErrorCodeConflict, message)
}

func Internal(message string) *GraphQLError {
	if message == "" {
		message = "Internal server error"
	}
	return NewError(ErrorCodeInternal, message)
}

func InternalWithError(err error) *GraphQLError {
	gqlErr := Internal("Internal server error")
	return gqlErr.WithExtension("originalError", err.Error())
}

func RateLimit(message string) *GraphQLError {
	if message == "" {
		message = "Rate limit exceeded"
	}
	return NewError(ErrorCodeRateLimit, message)
}

func WrapError(err error) error {
	if err == nil {
		return nil
	}

	if gqlErr, ok := err.(*GraphQLError); ok {
		return gqlErr
	}

	return InternalWithError(err)
}
