package service

import "errors"

// Repositories return these errors, optionally wrapped, for errors.Is classification.
var (
	ErrNotFound            = errors.New("service: record not found")
	ErrAlreadyExists       = errors.New("service: record already exists")
	ErrConcurrentUpdate    = errors.New("service: concurrent update")
	ErrTransactionRequired = errors.New("service: transaction required")
)
