package money

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

var (
	ErrInvalidCurrency  = errors.New("money: invalid currency")
	ErrInvalidAmount    = errors.New("money: invalid amount")
	ErrInvalidMoney     = errors.New("money: uninitialized value")
	ErrCurrencyMismatch = errors.New("money: currency mismatch")
	ErrOverflow         = errors.New("money: overflow")
)

type Money struct {
	amount   int64
	currency string
}

func New(amount int64, currency string) (Money, error) {
	switch currency {
	case "BRL", "USD":
		return Money{
			amount:   amount,
			currency: currency,
		}, nil
	default:
		return Money{}, ErrInvalidCurrency
	}
}

func Parse(amount string, currency string) (Money, error) {
	whole, fraction, found := strings.Cut(amount, ".")

	if !found || whole == "" || len(fraction) != 2 {
		return Money{}, ErrInvalidAmount
	}

	if len(whole) > 1 && whole[0] == '0' {
		return Money{}, ErrInvalidAmount
	}

	digits := whole + fraction

	for i := 0; i < len(digits); i++ {
		if digits[i] < '0' || digits[i] > '9' {
			return Money{}, ErrInvalidAmount
		}
	}

	minorUnits, err := strconv.ParseInt(digits, 10, 64)
	if err != nil {
		return Money{}, ErrInvalidAmount
	}

	return New(minorUnits, currency)

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

func Zero(currency string) (Money, error) {
	return New(0, currency)
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

func (m Money) String() string {
	if m.currency == "" {
		return "<invalid money>"
	}

	sign := ""
	magnitude := uint64(m.amount)

	if m.amount < 0 {
		sign = "-"
		magnitude = uint64(-(m.amount + 1)) + 1
	}

	return fmt.Sprintf(
		"%s%d.%02d",
		sign,
		magnitude/100,
		magnitude%100,
	)
}

func (m Money) MarshalJSON() ([]byte, error) {
	if m.currency == "" {
		return nil, ErrInvalidMoney
	}

	return json.Marshal(struct {
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
	}{
		Amount:   m.String(),
		Currency: m.currency,
	})
}

func (m Money) Amount() int64 {
	return m.amount
}

func (m Money) Currency() string {
	return m.currency
}
