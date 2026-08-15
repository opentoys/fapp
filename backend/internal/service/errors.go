package service

import "net/http"

// Error carries an HTTP-style status code plus a message. Controllers unwrap
// it to pick the response code; anything else is treated as a 500.
type Error struct {
	Status int
	Msg    string
}

func (e *Error) Error() string { return e.Msg }

// Status aliases to net/http status constants (untyped ints).
const (
	StatusBadRequest   = http.StatusBadRequest
	StatusUnauthorized = http.StatusUnauthorized
	StatusForbidden    = http.StatusForbidden
	StatusNotFound     = http.StatusNotFound
	StatusInternal     = http.StatusInternalServerError
)