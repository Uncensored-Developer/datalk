package errors

import "errors"

var (
	ErrFailedToAcquireLock   = errors.New("failed to acquire lock")
	ErrResourceAlreadyLocked = errors.New("resource already locked")
)
