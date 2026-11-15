package raidzero

import "fmt"

// Error is returned by every client method on a non-2xx API response.
type Error struct {
	// Code is the machine-readable error code from the API (e.g. "not_found").
	Code string
	// Message is the human-readable description.
	Message string
	// Status is the HTTP status code.
	Status int
}

func (e *Error) Error() string {
	return fmt.Sprintf("raidzero: %s (HTTP %d): %s", e.Code, e.Status, e.Message)
}

// PollTimeoutError is returned by PollTask when the timeout elapses.
type PollTimeoutError struct {
	TaskID    string
	TimeoutMs int64
}

func (e *PollTimeoutError) Error() string {
	return fmt.Sprintf("raidzero: task %s did not complete within %dms", e.TaskID, e.TimeoutMs)
}
