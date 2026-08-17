package service

// Error carries an HTTP-style status code plus a message. Controllers unwrap
// it to pick the response code; anything else is treated as a 500.
type Error struct {
	Status int
	Msg    string
}

func (e *Error) Error() string { return e.Msg }
