package errors

import "errors"

var (
	ErrBelowMOQ          = errors.New("quantity below minimum order quantity")
	ErrInvalidTransition = errors.New("invalid purchase order status transition")
	ErrArrivalExceeds    = errors.New("cumulative arrivals exceed ordered quantity")
	ErrForbidden         = errors.New("forbidden")
)
