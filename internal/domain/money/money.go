package money

import "math"

type Money struct {
	amount   int64
	currency string
}

func (m Money) Add(other Money) (Money, error) {
	if m.currency == "" || other.currency == "" {
		return Money{}, ErrInvalidMoney
	}

	if m.currency != other.currency {
		return Money{}, ErrCurrencyMismatch
	}

	if other.amount > 0 && m.amount > math.MaxInt64-other.amount {
		return Money{}, ErrOverflow
	}

	if other.amount < 0 && m.amount < math.MinInt64-other.amount {
		return Money{}, ErrOverflow
	}

	return New(m.amount+other.amount, m.currency)
}

func (m Money) Sub(other Money) (Money, error) {
	if m.currency == "" || other.currency == "" {
		return Money{}, ErrInvalidMoney
	}

	if m.currency != other.currency {
		return Money{}, ErrCurrencyMismatch
	}

	if other.amount > 0 && m.amount < math.MinInt64+other.amount {
		return Money{}, ErrOverflow
	}

	if other.amount < 0 && m.amount > math.MaxInt64+other.amount {
		return Money{}, ErrOverflow
	}

	return New(m.amount-other.amount, m.currency)
}

func (m Money) Neg() (Money, error) {
	if m.currency == "" {
		return Money{}, ErrInvalidMoney
	}

	if m.amount == math.MinInt64 {
		return Money{}, ErrOverflow
	}

	return New(-m.amount, m.currency)
}

func (m Money) Compare(other Money) (int, error) {
	if m.currency == "" || other.currency == "" {
		return 0, ErrInvalidMoney
	}

	if m.currency != other.currency {
		return 0, ErrCurrencyMismatch
	}

	switch {
	case m.amount < other.amount:
		return -1, nil
	case m.amount > other.amount:
		return 1, nil
	default:
		return 0, nil
	}
}

func (m Money) Amount() int64 {
	return m.amount
}

func (m Money) Currency() string {
	return m.currency
}
