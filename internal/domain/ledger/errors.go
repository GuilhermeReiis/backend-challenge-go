package ledger

import "errors"

var (
	ErrInvalidIdentity     = errors.New("ledger: invalid identity")
	ErrInvalidDirection    = errors.New("ledger: invalid direction")
	ErrInvalidAmount       = errors.New("ledger: amount must be positive")
	ErrNegativeBalance     = errors.New("ledger: negative balance")
	ErrInconsistentBalance = errors.New("ledger: inconsistent balance")
	ErrInvalidTime         = errors.New("ledger: invalid time")
)
