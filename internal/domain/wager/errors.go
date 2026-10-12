package wager

import "errors"

var (
	ErrInvalidIdentity    = errors.New("wager: invalid identity")
	ErrInvalidKind        = errors.New("wager: invalid external kind")
	ErrInvalidAmount      = errors.New("wager: invalid amount for kind")
	ErrInvalidReference   = errors.New("wager: invalid reference")
	ErrInvalidTime        = errors.New("wager: invalid time")
	ErrInvalidTransaction = errors.New("wager: uninitialized transaction")
	ErrTerminalState      = errors.New("wager: terminal state")
	ErrInvalidFailureCode = errors.New("wager: invalid failure code")
)
