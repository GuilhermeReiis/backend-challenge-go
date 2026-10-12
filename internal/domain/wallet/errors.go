package wallet

import "errors"

var (
	ErrInvalidIdentity   = errors.New("wallet: invalid identity")
	ErrNegativeBalance   = errors.New("wallet: negative balance")
	ErrInvalidTime       = errors.New("wallet: invalid time")
	ErrInvalidVersion    = errors.New("wallet: invalid version")
	ErrInvalidWallet     = errors.New("wallet: uninitialized value")
	ErrInvalidMovement   = errors.New("wallet: negative movement")
	ErrInsufficientFunds = errors.New("wallet: insufficient funds")
	ErrVersionOverflow   = errors.New("wallet: version overflow")
)
