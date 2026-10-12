package money

import "errors"

var (
	ErrInvalidCurrency  = errors.New("money: invalid currency")
	ErrInvalidAmount    = errors.New("money: invalid amount")
	ErrInvalidMoney     = errors.New("money: uninitialized value")
	ErrCurrencyMismatch = errors.New("money: currency mismatch")
	ErrOverflow         = errors.New("money: overflow")
)
