package errors

import "errors"

var (
	ErrNotFound          = errors.New("resource not found")
	ErrInvalidInput      = errors.New("invalid request input")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrInvalidTransition = errors.New("invalid purchase order status transition")
	ErrExceedsOrdered    = errors.New("cumulative deliveries exceed ordered quantity")
)

type BusinessError struct {
	Code    int
	Message string
	Err     error
}

func (e *BusinessError) Error() string { return e.Message }
func (e *BusinessError) Unwrap() error { return e.Err }

func NewBusiness(code int, message string) *BusinessError {
	return &BusinessError{Code: code, Message: message}
}
