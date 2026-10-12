package ledger_test

import (
	"errors"
	"testing"
	"time"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/ledger"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
)

func newMoney(t *testing.T, amount int64, currency string) money.Money {
	t.Helper()

	value, err := money.New(amount, currency)

	if err != nil {
		t.Fatalf("failed to create test money: %v", err)
	}

	return value
}

func TestNew(t *testing.T) {
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

	t.Run("creates consistent entries", func(t *testing.T) {
		cases := []struct {
			name      string
			direction ledger.Direction
			before    int64
			amount    int64
			after     int64
		}{
			{
				name:      "debit",
				direction: ledger.Debit,
				before:    10000, amount: 2500, after: 7500,
			},
			{
				name:      "credit",
				direction: ledger.Credit,
				before:    10000, amount: 2500, after: 12500,
			},
			{
				name:      "opening credit",
				direction: ledger.Credit,
				before:    0, amount: 10000, after: 10000,
			},
			{
				name:      "debit to zero",
				direction: ledger.Debit,
				before:    10000, amount: 10000, after: 0,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				value, err := ledger.New(
					"entry-1",
					"wallet-1",
					"transaction-1",
					tc.direction,
					newMoney(t, tc.amount, "BRL"),
					newMoney(t, tc.before, "BRL"),
					newMoney(t, tc.after, "BRL"),
					now,
				)

				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if value.ID() != "entry-1" ||
					value.WalletID() != "wallet-1" ||
					value.TransactionID() != "transaction-1" {
					t.Error("expected identities to be preserved")
				}

				if value.Direction() != tc.direction {
					t.Errorf("expected direction %s, got %s",
						tc.direction, value.Direction())
				}

				if value.Money().Amount() != tc.amount {
					t.Errorf("expected amount %d, got %d",
						tc.amount, value.Money().Amount())
				}

				if value.BalanceBefore().Amount() != tc.before ||
					value.BalanceAfter().Amount() != tc.after {
					t.Error("expected balances to be preserved")
				}

				if !value.CreatedAt().Equal(now) {
					t.Error("expected creation timestamp to be preserved")
				}
			})
		}
	})

	t.Run("rejects invalid entries", func(t *testing.T) {
		cases := []struct {
			name      string
			direction ledger.Direction
			amount    money.Money
			before    money.Money
			after     money.Money
			wantErr   error
		}{
			{
				name:      "incorrect resulting balance",
				direction: ledger.Debit,
				amount:    newMoney(t, 2500, "BRL"),
				before:    newMoney(t, 10000, "BRL"),
				after:     newMoney(t, 8000, "BRL"),
				wantErr:   ledger.ErrInconsistentBalance,
			},
			{
				name:      "invalid direction",
				direction: ledger.Direction("UNKNOWN"),
				amount:    newMoney(t, 2500, "BRL"),
				before:    newMoney(t, 10000, "BRL"),
				after:     newMoney(t, 7500, "BRL"),
				wantErr:   ledger.ErrInvalidDirection,
			},
			{
				name:      "zero amount",
				direction: ledger.Debit,
				amount:    newMoney(t, 0, "BRL"),
				before:    newMoney(t, 10000, "BRL"),
				after:     newMoney(t, 10000, "BRL"),
				wantErr:   ledger.ErrInvalidAmount,
			},
			{
				name:      "negative resulting balance",
				direction: ledger.Debit,
				amount:    newMoney(t, 15000, "BRL"),
				before:    newMoney(t, 10000, "BRL"),
				after:     newMoney(t, -5000, "BRL"),
				wantErr:   ledger.ErrNegativeBalance,
			},
			{
				name:      "incompatible movement currency",
				direction: ledger.Debit,
				amount:    newMoney(t, 2500, "USD"),
				before:    newMoney(t, 10000, "BRL"),
				after:     newMoney(t, 7500, "BRL"),
				wantErr:   money.ErrCurrencyMismatch,
			},
			{
				name:      "incompatible resulting currency",
				direction: ledger.Debit,
				amount:    newMoney(t, 2500, "BRL"),
				before:    newMoney(t, 10000, "BRL"),
				after:     newMoney(t, 7500, "USD"),
				wantErr:   money.ErrCurrencyMismatch,
			},
			{
				name:      "uninitialized amount",
				direction: ledger.Debit,
				before:    newMoney(t, 10000, "BRL"),
				after:     newMoney(t, 7500, "BRL"),
				wantErr:   money.ErrInvalidMoney,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if _, err := ledger.New(
					"entry-1", "wallet-1", "transaction-1",
					tc.direction, tc.amount, tc.before, tc.after, now,
				); !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got: %v", tc.wantErr, err)
				}
			})
		}
	})
}
