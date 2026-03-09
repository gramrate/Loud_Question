package errorz

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrForbidden     = errors.New("forbidden")
	ErrAlreadyExists = errors.New("already exists")
	ErrConflict      = errors.New("conflict")
	ErrLimitExceeded = errors.New("limit exceeded")
)
