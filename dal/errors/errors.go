package errors

import (
	"errors"
	"fmt"
)

// Common errors
var (
	ErrNotFound      = errors.New("record not found")
	ErrDuplicate     = errors.New("duplicate record")
	ErrConstraint    = errors.New("constraint violation")
	ErrConnection    = errors.New("connection error")
	ErrTimeout       = errors.New("operation timeout")
	ErrInvalidInput  = errors.New("invalid input")
	ErrTransaction   = errors.New("transaction error")
)

// ErrorCode represents different types of database errors
type ErrorCode string

const (
	CodeNotFound      ErrorCode = "NOT_FOUND"
	CodeDuplicate     ErrorCode = "DUPLICATE"
	CodeConstraint    ErrorCode = "CONSTRAINT"
	CodeConnection    ErrorCode = "CONNECTION"
	CodeTimeout       ErrorCode = "TIMEOUT"
	CodeInvalidInput  ErrorCode = "INVALID_INPUT"
	CodeTransaction   ErrorCode = "TRANSACTION"
	CodeValidation    ErrorCode = "VALIDATION"
	CodeUnknown       ErrorCode = "UNKNOWN"
)

// Error represents a database error with additional context
type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
	Details map[string]interface{}
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *Error) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target
func (e *Error) Is(target error) bool {
	if e.Cause != nil {
		return errors.Is(e.Cause, target)
	}
	return false
}

// NewError creates a new Error
func NewError(code ErrorCode, message string, cause error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
		Details: make(map[string]interface{}),
	}
}

// NewNotFoundError creates a not found error
func NewNotFoundError(entity string, id string) *Error {
	return &Error{
		Code:    CodeNotFound,
		Message: fmt.Sprintf("%s with ID %s not found", entity, id),
		Details: map[string]interface{}{
			"entity": entity,
			"id":     id,
		},
	}
}

// NewDuplicateError creates a duplicate error
func NewDuplicateError(entity string, field string, value interface{}) *Error {
	return &Error{
		Code:    CodeDuplicate,
		Message: fmt.Sprintf("%s with %s '%v' already exists", entity, field, value),
		Details: map[string]interface{}{
			"entity": entity,
			"field":  field,
			"value":  value,
		},
	}
}

// NewConstraintError creates a constraint violation error
func NewConstraintError(constraint string, cause error) *Error {
	return &Error{
		Code:    CodeConstraint,
		Message: fmt.Sprintf("constraint violation: %s", constraint),
		Cause:   cause,
		Details: map[string]interface{}{
			"constraint": constraint,
		},
	}
}

// IsNotFound checks if the error is a not found error
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Code == CodeNotFound
	}
	return errors.Is(err, ErrNotFound)
}

// IsDuplicate checks if the error is a duplicate error
func IsDuplicate(err error) bool {
	if err == nil {
		return false
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Code == CodeDuplicate
	}
	return errors.Is(err, ErrDuplicate)
}

// WrapError wraps an error with additional context
func WrapError(err error, message ...string) error {
	if err == nil {
		return nil
	}
	
	msg := "wrapped error"
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}
	
	// If it's already our Error type, preserve the code
	var e *Error
	if errors.As(err, &e) {
		return &Error{
			Code:    e.Code,
			Message: msg,
			Cause:   err,
			Details: e.Details,
		}
	}
	
	// Otherwise create a new error with unknown code
	return &Error{
		Code:    CodeUnknown,
		Message: msg,
		Cause:   err,
		Details: make(map[string]interface{}),
	}
}