package models

import (
	"fmt"
	"net/http"
)

// ErrorCode represents different types of errors in the system
type ErrorCode string

const (
	// Client errors (4xx)
	ErrCodeBadRequest       ErrorCode = "BAD_REQUEST"
	ErrCodeUnauthorized     ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden        ErrorCode = "FORBIDDEN"
	ErrCodeNotFound         ErrorCode = "NOT_FOUND"
	ErrCodeConflict         ErrorCode = "CONFLICT"
	ErrCodeValidation       ErrorCode = "VALIDATION_ERROR"
	ErrCodeRateLimit        ErrorCode = "RATE_LIMIT_EXCEEDED"
	
	// Server errors (5xx)
	ErrCodeInternal         ErrorCode = "INTERNAL_ERROR"
	ErrCodeDatabase         ErrorCode = "DATABASE_ERROR"
	ErrCodeCache            ErrorCode = "CACHE_ERROR"
	ErrCodeExternal         ErrorCode = "EXTERNAL_SERVICE_ERROR"
)

// AppError represents an application error with context
type AppError struct {
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	Details    string    `json:"details,omitempty"`
	HTTPStatus int       `json:"-"`
	Cause      error     `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause
func (e *AppError) Unwrap() error {
	return e.Cause
}

// NewAppError creates a new application error
func NewAppError(code ErrorCode, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// NewAppErrorWithCause creates a new application error with a cause
func NewAppErrorWithCause(code ErrorCode, message string, httpStatus int, cause error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Cause:      cause,
	}
}

// NewAppErrorWithDetails creates a new application error with details
func NewAppErrorWithDetails(code ErrorCode, message, details string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		Details:    details,
		HTTPStatus: httpStatus,
	}
}

// Common error constructors
func ErrBadRequest(message string) *AppError {
	return NewAppError(ErrCodeBadRequest, message, http.StatusBadRequest)
}

func ErrUnauthorized(message string) *AppError {
	return NewAppError(ErrCodeUnauthorized, message, http.StatusUnauthorized)
}

func ErrForbidden(message string) *AppError {
	return NewAppError(ErrCodeForbidden, message, http.StatusForbidden)
}

func ErrNotFound(message string) *AppError {
	return NewAppError(ErrCodeNotFound, message, http.StatusNotFound)
}

func ErrConflict(message string) *AppError {
	return NewAppError(ErrCodeConflict, message, http.StatusConflict)
}

func ErrValidation(message string) *AppError {
	return NewAppError(ErrCodeValidation, message, http.StatusBadRequest)
}

func ErrRateLimit(message string) *AppError {
	return NewAppError(ErrCodeRateLimit, message, http.StatusTooManyRequests)
}

func ErrInternal(message string) *AppError {
	return NewAppError(ErrCodeInternal, message, http.StatusInternalServerError)
}

func ErrDatabase(message string, cause error) *AppError {
	return NewAppErrorWithCause(ErrCodeDatabase, message, http.StatusInternalServerError, cause)
}

func ErrCache(message string, cause error) *AppError {
	return NewAppErrorWithCause(ErrCodeCache, message, http.StatusInternalServerError, cause)
}

func ErrExternal(message string, cause error) *AppError {
	return NewAppErrorWithCause(ErrCodeExternal, message, http.StatusBadGateway, cause)
}

// Specific domain errors
var (
	ErrURLNotFound = ErrNotFound("URL not found")
	ErrURLExpired  = ErrNotFound("URL has expired")
	ErrURLInactive = ErrNotFound("URL is inactive")
	
	ErrInvalidURL        = ErrValidation("Invalid URL format")
	ErrURLTooLong        = ErrValidation("URL is too long")
	ErrInvalidShortCode  = ErrValidation("Invalid short code format")
	ErrInvalidCustomAlias = ErrValidation("Invalid custom alias")
	
	ErrShortCodeExists    = ErrConflict("Short code already exists")
	ErrCustomAliasExists  = ErrConflict("Custom alias already exists")
	
	ErrRateLimitExceeded = ErrRateLimit("Rate limit exceeded")
)
