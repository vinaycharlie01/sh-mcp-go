package errors

import (
	"errors"
	"fmt"
)

// Sentinel domain errors.
var (
	ErrNotFound          = errors.New("not found")
	ErrAlreadyExists     = errors.New("already exists")
	ErrInvalidInput      = errors.New("invalid input")
	ErrConflict          = errors.New("conflict")
	ErrTimeout           = errors.New("timeout")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrForbidden         = errors.New("forbidden")
	ErrInternalServer    = errors.New("internal server error")
	ErrUnavailable       = errors.New("service unavailable")
	ErrValidationFailed  = errors.New("validation failed")
	ErrDeploymentFailed  = errors.New("deployment failed")
	ErrRollbackFailed    = errors.New("rollback failed")
	ErrUpgradeFailed     = errors.New("upgrade failed")
	ErrHealthCheckFailed = errors.New("health check failed")
)

// DomainError carries structured context for a domain error.
type DomainError struct {
	Code    string
	Message string
	Cause   error
	Fields  map[string]any
}

func (e *DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *DomainError) Unwrap() error { return e.Cause }

// New wraps a sentinel error with a descriptive message.
func New(sentinel error, msg string, args ...any) *DomainError {
	return &DomainError{
		Code:    codeFor(sentinel),
		Message: fmt.Sprintf(msg, args...),
		Cause:   sentinel,
	}
}

// Wrap adds context to an existing error.
func Wrap(cause error, msg string, args ...any) *DomainError {
	return &DomainError{
		Code:    "WRAPPED",
		Message: fmt.Sprintf(msg, args...),
		Cause:   cause,
	}
}

// WithFields attaches structured metadata to a DomainError.
func WithFields(err *DomainError, fields map[string]any) *DomainError {
	err.Fields = fields
	return err
}

// Is delegates to stdlib errors.Is.
func Is(err, target error) bool { return errors.Is(err, target) }

// As delegates to stdlib errors.As.
func As(err error, target any) bool { return errors.As(err, target) }

func codeFor(err error) string {
	switch {
	case errors.Is(err, ErrNotFound):
		return "NOT_FOUND"
	case errors.Is(err, ErrAlreadyExists):
		return "ALREADY_EXISTS"
	case errors.Is(err, ErrInvalidInput):
		return "INVALID_INPUT"
	case errors.Is(err, ErrConflict):
		return "CONFLICT"
	case errors.Is(err, ErrTimeout):
		return "TIMEOUT"
	case errors.Is(err, ErrUnauthorized):
		return "UNAUTHORIZED"
	case errors.Is(err, ErrForbidden):
		return "FORBIDDEN"
	case errors.Is(err, ErrValidationFailed):
		return "VALIDATION_FAILED"
	case errors.Is(err, ErrDeploymentFailed):
		return "DEPLOYMENT_FAILED"
	case errors.Is(err, ErrRollbackFailed):
		return "ROLLBACK_FAILED"
	case errors.Is(err, ErrUpgradeFailed):
		return "UPGRADE_FAILED"
	case errors.Is(err, ErrHealthCheckFailed):
		return "HEALTH_CHECK_FAILED"
	default:
		return "INTERNAL"
	}
}
