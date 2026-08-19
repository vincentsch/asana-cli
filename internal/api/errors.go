// Package api defines API errors and exit codes.
package api

import (
	"fmt"
	"net/http"
)

const (
	ExitSuccess    = 0
	ExitUsageError = 1
	ExitAPIError   = 2
)

// APIError captures structured Asana API errors.
type APIError struct {
	StatusCode int
	RequestID  string
	Errors     []AsanaError
}

func (e *APIError) Error() string {
	message := http.StatusText(e.StatusCode)
	if len(e.Errors) > 0 && e.Errors[0].Message != "" {
		message = e.Errors[0].Message
	}

	base := fmt.Sprintf("API error (HTTP %d): %s", e.StatusCode, message)
	if e.RequestID == "" {
		return base
	}

	return fmt.Sprintf("%s\nRequest ID: %s", base, e.RequestID)
}

// IsUnauthorized reports whether the error is an HTTP 401.
func (e *APIError) IsUnauthorized() bool {
	if e == nil {
		return false
	}
	return e.StatusCode == http.StatusUnauthorized
}

// IsPremiumRequired reports whether the error is an HTTP 402 (Payment Required).
func (e *APIError) IsPremiumRequired() bool {
	if e == nil {
		return false
	}
	return e.StatusCode == http.StatusPaymentRequired
}

// ResponseError wraps unexpected API response failures.
type ResponseError struct {
	Err error
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("API response error: %v", e.Err)
}

func (e *ResponseError) Unwrap() error {
	return e.Err
}
