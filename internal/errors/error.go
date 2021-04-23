package errors

import "fmt"

// Error encapsulates error info.
type Error interface {
	withMessage(msg string) Error
	AddMeta(key string, value interface{}) Error
	getCode() code
	Error() string
	Message() string
	ExitStatus() int
}

// nrError encapsulates error info.
type nrError struct {
	code       code
	msg        string
	meta       map[string]interface{}
	exitStatus int
}

func createError(code code, exitStatus int) Error {
	return &nrError{
		code:       code,
		exitStatus: exitStatus,
	}
}

// Error converts the error into a readable message.
func (e *nrError) Error() string {
	out := fmt.Sprintf("[ERR] %s", e.msg)
	if e.meta != nil {
		for k, v := range e.meta {
			out += fmt.Sprintf("\n  - %s: %v", k, v)
		}
	}
	return out
}

func (e *nrError) getCode() code {
	return e.code
}

func (e *nrError) Message() string {
	return e.msg
}

// withMessage add the message to the error.
func (e *nrError) withMessage(msg string) Error {
	e.msg = msg
	return e
}

// AddMeta permits add extra info to show displayed when printing the error.
func (e *nrError) AddMeta(key string, value interface{}) Error {
	if e.meta == nil {
		e.meta = make(map[string]interface{})
	}
	e.meta[key] = value

	return e
}

func (e *nrError) ExitStatus() int {
	return e.exitStatus
}
