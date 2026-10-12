package money_test

import (
	"encoding/json"
	"errors"
	"math"
	"testing"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
)

func TestNew(t *testing.T) {
	t.Run("creates money with valid amount and currency", func(t *testing.T) {
		value, err := money.New(2500, "BRL")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if value.Amount() != 2500 {
			t.Errorf("expected amount 2500, got %d", value.Amount())
		}

		if value.Currency() != "BRL" {
			t.Errorf("expected currency BRL, got %s", value.Currency())
		}
	})

	t.Run("rejects invalid currency", func(t *testing.T) {
		if _, err := money.New(2500, ""); !errors.Is(err, money.ErrInvalidCurrency) {
			t.Fatalf("expected invalid currency error, got: %v", err)
		}
	})
}

func TestParse(t *testing.T) {
	t.Run("converts decimal amount to minor units", func(t *testing.T) {
		value, err := money.Parse("25.00", "BRL")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if value.Amount() != 2500 {
			t.Errorf("expected amount 2500, got %d", value.Amount())
		}

		if value.Currency() != "BRL" {
			t.Errorf("expected currency BRL, got %s", value.Currency())
		}
	})

	t.Run("rejects invalid amounts", func(t *testing.T) {
		cases := []struct {
			name   string
			amount string
		}{
			{name: "empty amount", amount: ""},
			{name: "missing decimal places", amount: "25"},
			{name: "one decimal place", amount: "25.0"},
			{name: "excess decimal places", amount: "25.001"},
			{name: "negative amount", amount: "-25.00"},
			{name: "leading zero", amount: "025.00"},
			{name: "NaN", amount: "NaN"},
			{name: "infinity", amount: "Infinity"},
			{name: "scientific notation", amount: "1e2"},
			{name: "whitespace", amount: "25. 0"},
			{name: "overflow", amount: "92233720368547758.08"},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if _, err := money.Parse(tc.amount, "BRL"); !errors.Is(err, money.ErrInvalidAmount) {
					t.Fatalf("expected invalid amount error, got: %v", err)
				}
			})
		}
	})
}

func TestAdd(t *testing.T) {
	t.Run("adds amounts without changing operands", func(t *testing.T) {
		left, err := money.New(2500, "BRL")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		right, err := money.New(1000, "BRL")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := left.Add(right)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Amount() != 3500 {
			t.Errorf("expected amount 3500, got %d", result.Amount())
		}

		if result.Currency() != "BRL" {
			t.Errorf("expected currency BRL, got %s", result.Currency())
		}

		if left.Amount() != 2500 || right.Amount() != 1000 {
			t.Error("expected operands to remain unchanged")
		}
	})

	t.Run("rejects incompatible currencies", func(t *testing.T) {
		left, err := money.New(2500, "BRL")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		right, err := money.New(1000, "USD")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, err := left.Add(right); !errors.Is(err, money.ErrCurrencyMismatch) {
			t.Fatalf("expected currency mismatch error, got: %v", err)
		}
	})

	t.Run("rejects uninitialized money", func(t *testing.T) {
		valid, err := money.New(1000, "BRL")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cases := []struct {
			name  string
			left  money.Money
			right money.Money
		}{
			{name: "left operand", left: money.Money{}, right: valid},
			{name: "right operand", left: valid, right: money.Money{}},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if _, err := tc.left.Add(tc.right); !errors.Is(err, money.ErrInvalidMoney) {
					t.Fatalf("expected invalid money error, got: %v", err)
				}
			})
		}
	})

	t.Run("rejects overflow", func(t *testing.T) {
		cases := []struct {
			name  string
			left  int64
			right int64
		}{
			{name: "upper limit", left: math.MaxInt64, right: 1},
			{name: "lower limit", left: math.MinInt64, right: -1},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				left, err := money.New(tc.left, "BRL")

				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				right, err := money.New(tc.right, "BRL")

				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if _, err := left.Add(right); !errors.Is(err, money.ErrOverflow) {
					t.Fatalf("expected overflow error, got: %v", err)
				}
			})
		}
	})
}

func TestSub(t *testing.T) {
	t.Run("subtracts amounts without changing operands", func(t *testing.T) {
		left, err := money.New(2500, "BRL")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		right, err := money.New(1000, "BRL")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := left.Sub(right)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Amount() != 1500 {
			t.Errorf("expected amount 1500, got %d", result.Amount())
		}

		if result.Currency() != "BRL" {
			t.Errorf("expected currency BRL, got %s", result.Currency())
		}

		if left.Amount() != 2500 || right.Amount() != 1000 {
			t.Error("expected operands to remain unchanged")
		}
	})

	t.Run("allows negative results", func(t *testing.T) {
		left, err := money.New(1000, "BRL")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		right, err := money.New(2500, "BRL")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := left.Sub(right)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Amount() != -1500 {
			t.Errorf("expected amount -1500, got %d", result.Amount())
		}
	})

	t.Run("rejects incompatible currencies", func(t *testing.T) {
		left, err := money.New(2500, "BRL")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		right, err := money.New(1000, "USD")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, err := left.Sub(right); !errors.Is(err, money.ErrCurrencyMismatch) {
			t.Fatalf("expected currency mismatch error, got: %v", err)
		}
	})

	t.Run("rejects uninitialized money", func(t *testing.T) {
		valid, err := money.New(1000, "BRL")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cases := []struct {
			name  string
			left  money.Money
			right money.Money
		}{
			{name: "left operand", left: money.Money{}, right: valid},
			{name: "right operand", left: valid, right: money.Money{}},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if _, err := tc.left.Sub(tc.right); !errors.Is(err, money.ErrInvalidMoney) {
					t.Fatalf("expected invalid money error, got: %v", err)
				}
			})
		}
	})

	t.Run("rejects overflow", func(t *testing.T) {
		cases := []struct {
			name  string
			left  int64
			right int64
		}{
			{name: "lower limit", left: math.MinInt64, right: 1},
			{name: "upper limit", left: math.MaxInt64, right: -1},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				left, err := money.New(tc.left, "BRL")

				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				right, err := money.New(tc.right, "BRL")

				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if _, err := left.Sub(right); !errors.Is(err, money.ErrOverflow) {
					t.Fatalf("expected overflow error, got: %v", err)
				}
			})
		}
	})
}

func TestZero(t *testing.T) {
	t.Run("creates zero with the requested currency", func(t *testing.T) {
		value, err := money.Zero("BRL")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if value.Amount() != 0 {
			t.Errorf("expected amount 0, got %d", value.Amount())
		}

		if value.Currency() != "BRL" {
			t.Errorf("expected currency BRL, got %s", value.Currency())
		}
	})

	t.Run("rejects invalid currency", func(t *testing.T) {
		if _, err := money.Zero(""); !errors.Is(err, money.ErrInvalidCurrency) {
			t.Fatalf("expected invalid currency error, got: %v", err)
		}
	})
}

func TestNeg(t *testing.T) {
	t.Run("negates without changing the original", func(t *testing.T) {
		cases := []struct {
			name   string
			amount int64
			want   int64
		}{
			{name: "positive", amount: 2500, want: -2500},
			{name: "negative", amount: -2500, want: 2500},
			{name: "zero", amount: 0, want: 0},
			{name: "maximum", amount: math.MaxInt64, want: -math.MaxInt64},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				original := newMoney(t, tc.amount, "BRL")

				result, err := original.Neg()

				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if result.Amount() != tc.want {
					t.Errorf("expected amount %d, got %d", tc.want, result.Amount())
				}

				if result.Currency() != "BRL" {
					t.Errorf("expected currency BRL, got %s", result.Currency())
				}

				if original.Amount() != tc.amount {
					t.Error("expected original to remain unchanged")
				}
			})
		}
	})

	t.Run("rejects overflow", func(t *testing.T) {
		value := newMoney(t, math.MinInt64, "BRL")

		if _, err := value.Neg(); !errors.Is(err, money.ErrOverflow) {
			t.Fatalf("expected overflow error, got: %v", err)
		}
	})

	t.Run("rejects uninitialized money", func(t *testing.T) {
		if _, err := (money.Money{}).Neg(); !errors.Is(err, money.ErrInvalidMoney) {
			t.Fatalf("expected invalid money error, got: %v", err)
		}
	})
}

func TestCompare(t *testing.T) {
	t.Run("compares amounts with the same currency", func(t *testing.T) {
		cases := []struct {
			name  string
			left  int64
			right int64
			want  int
		}{
			{name: "less", left: 1000, right: 2500, want: -1},
			{name: "equal", left: 2500, right: 2500, want: 0},
			{name: "greater", left: 2500, right: 1000, want: 1},
			{name: "negative", left: -2500, right: 0, want: -1},
			{name: "numeric limits", left: math.MinInt64, right: math.MaxInt64, want: -1},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				left := newMoney(t, tc.left, "BRL")
				right := newMoney(t, tc.right, "BRL")

				result, err := left.Compare(right)

				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if result != tc.want {
					t.Errorf("expected comparison %d, got %d", tc.want, result)
				}
			})
		}
	})

	t.Run("rejects incompatible currencies", func(t *testing.T) {
		left := newMoney(t, 2500, "BRL")
		right := newMoney(t, 2500, "USD")

		if _, err := left.Compare(right); !errors.Is(err, money.ErrCurrencyMismatch) {
			t.Fatalf("expected currency mismatch error, got: %v", err)
		}
	})

	t.Run("rejects uninitialized money", func(t *testing.T) {
		valid := newMoney(t, 2500, "BRL")

		cases := []struct {
			name  string
			left  money.Money
			right money.Money
		}{
			{name: "left operand", left: money.Money{}, right: valid},
			{name: "right operand", left: valid, right: money.Money{}},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if _, err := tc.left.Compare(tc.right); !errors.Is(err, money.ErrInvalidMoney) {
					t.Fatalf("expected invalid money error, got: %v", err)
				}
			})
		}
	})
}

func TestString(t *testing.T) {
	t.Run("formats amounts with two decimal places", func(t *testing.T) {
		cases := []struct {
			name   string
			amount int64
			want   string
		}{
			{name: "zero", amount: 0, want: "0.00"},
			{name: "one cent", amount: 1, want: "0.01"},
			{name: "whole amount", amount: 2500, want: "25.00"},
			{name: "fractional amount", amount: 2505, want: "25.05"},
			{name: "negative amount", amount: -2505, want: "-25.05"},
			{name: "maximum", amount: math.MaxInt64, want: "92233720368547758.07"},
			{name: "minimum", amount: math.MinInt64, want: "-92233720368547758.08"},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				value := newMoney(t, tc.amount, "BRL")

				if got := value.String(); got != tc.want {
					t.Errorf("expected %q, got %q", tc.want, got)
				}
			})
		}
	})

	t.Run("identifies uninitialized money", func(t *testing.T) {
		value := money.Money{}

		if got := value.String(); got != "<invalid money>" {
			t.Errorf("expected invalid money marker, got %q", got)
		}
	})
}

func TestMarshalJSON(t *testing.T) {
	t.Run("serializes amount as a decimal string", func(t *testing.T) {
		cases := []struct {
			name   string
			amount int64
			want   string
		}{
			{
				name:   "positive",
				amount: 2500,
				want:   `{"amount":"25.00","currency":"BRL"}`,
			},
			{
				name:   "zero",
				amount: 0,
				want:   `{"amount":"0.00","currency":"BRL"}`,
			},
			{
				name:   "negative internal result",
				amount: -1500,
				want:   `{"amount":"-15.00","currency":"BRL"}`,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				value := newMoney(t, tc.amount, "BRL")

				result, err := json.Marshal(value)

				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if string(result) != tc.want {
					t.Errorf("expected %s, got %s", tc.want, result)
				}
			})
		}
	})

	t.Run("rejects uninitialized money", func(t *testing.T) {
		if _, err := json.Marshal(money.Money{}); !errors.Is(err, money.ErrInvalidMoney) {
			t.Fatalf("expected invalid money error, got: %v", err)
		}
	})
}

func newMoney(t *testing.T, amount int64, currency string) money.Money {
	t.Helper()

	value, err := money.New(amount, currency)

	if err != nil {
		t.Fatalf("failed to create test money: %v", err)
	}

	return value
}
