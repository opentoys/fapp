package server

type webErr struct {
	code int
	msg  string
}

func (e *webErr) Error() string { return e.msg }
