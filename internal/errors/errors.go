package errors

import (
	"fmt"
	"net/http"
)

// AppError represents an application error with HTTP status code.
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new AppError.
func New(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap wraps an error with an AppError.
func Wrap(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Common errors
var (
	ErrNotFound     = New(http.StatusNotFound, "resource not found")
	ErrBadRequest   = New(http.StatusBadRequest, "bad request")
	ErrUnauthorized = New(http.StatusUnauthorized, "unauthorized")
	ErrForbidden    = New(http.StatusForbidden, "forbidden")
	ErrInternal     = New(http.StatusInternalServerError, "internal server error")
	ErrConflict     = New(http.StatusConflict, "resource already exists")
)

// NotFound creates a not found error.
func NotFound(resource string) *AppError {
	return New(http.StatusNotFound, fmt.Sprintf("%s not found", resource))
}

// BadRequest creates a bad request error.
func BadRequest(message string) *AppError {
	return New(http.StatusBadRequest, message)
}

// Unauthorized creates an unauthorized error.
func Unauthorized(message string) *AppError {
	return New(http.StatusUnauthorized, message)
}

// Internal creates an internal server error.
func Internal(err error) *AppError {
	return Wrap(http.StatusInternalServerError, "internal server error", err)
}

// Validation creates a validation error.
func Validation(message string) *AppError {
	return New(http.StatusUnprocessableEntity, message)
}
