package money

import (
	"strconv"
	"strings"
)

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

func Zero(currency string) (Money, error) {
	return New(0, currency)
}
