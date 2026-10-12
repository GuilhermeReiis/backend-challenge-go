package wallet_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/wallet"
)

func newMoney(t *testing.T, amount int64, currency string) money.Money {
	t.Helper()

	value, err := money.New(amount, currency)

	if err != nil {
		t.Fatalf("failed to create test money: %v", err)
	}

	return value
}

func TestMovementTimestamps(t *testing.T) {
	createdAt := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(2 * time.Hour)
	amount := newMoney(t, 100, "BRL")

	movements := []struct {
		name  string
		apply func(*wallet.Wallet, money.Money, time.Time) error
	}{
		{name: "credit", apply: (*wallet.Wallet).Credit},
		{name: "debit", apply: (*wallet.Wallet).Debit},
	}

	for _, movement := range movements {
		t.Run(movement.name, func(t *testing.T) {
			cases := []struct {
				name    string
				now     time.Time
				wantErr error
			}{
				{
					name:    "rejects time before the last update",
					now:     createdAt.Add(time.Hour),
					wantErr: wallet.ErrInvalidTime,
				},
				{
					name: "accepts time equal to the last update",
					now:  updatedAt,
				},
			}

			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					value, err := wallet.New(
						"wallet-1", "player-1", newMoney(t, 10000, "BRL"), createdAt,
					)

					if err != nil {
						t.Fatal(err)
					}

					if err := value.Credit(amount, updatedAt); err != nil {
						t.Fatal(err)
					}

					before := *value

					if err := movement.apply(value, amount, tc.now); !errors.Is(err, tc.wantErr) {
						t.Fatalf("expected error %v, got: %v", tc.wantErr, err)
					}

					if tc.wantErr != nil {
						if *value != before {
							t.Error("expected rejected movement to preserve the complete state")
						}

						return
					}

					if value.Version() != before.Version()+1 ||
						!value.UpdatedAt().Equal(updatedAt) {
						t.Error("expected movement to increment the version and preserve the accepted timestamp")
					}
				})
			}
		})
	}
}

func TestNew(t *testing.T) {
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

	t.Run("creates wallet with initial version and timestamps", func(t *testing.T) {
		balance, err := money.Parse("100.00", "BRL")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		value, err := wallet.New("wallet-1", "player-1", balance, now)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if value.ID() != "wallet-1" {
			t.Errorf("unexpected wallet ID: %s", value.ID())
		}

		if value.PlayerID() != "player-1" {
			t.Errorf("unexpected player ID: %s", value.PlayerID())
		}

		if value.Balance().Amount() != 10000 {
			t.Errorf("expected balance 10000, got %d", value.Balance().Amount())
		}

		if value.Currency() != "BRL" {
			t.Errorf("expected currency BRL, got %s", value.Currency())
		}

		if value.Version() != 1 {
			t.Errorf("expected version 1, got %d", value.Version())
		}

		if !value.CreatedAt().Equal(now) || !value.UpdatedAt().Equal(now) {
			t.Error("expected creation and update timestamps to match now")
		}
	})

	t.Run("accepts zero initial balance", func(t *testing.T) {
		balance, err := money.Zero("BRL")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		value, err := wallet.New("wallet-1", "player-1", balance, now)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if value.Balance().Amount() != 0 {
			t.Errorf("expected zero balance, got %d", value.Balance().Amount())
		}
	})

	t.Run("rejects invalid creation data", func(t *testing.T) {
		validBalance, err := money.Zero("BRL")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		negativeBalance, err := money.New(-100, "BRL")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cases := []struct {
			name     string
			id       string
			playerID string
			balance  money.Money
			now      time.Time
			wantErr  error
		}{
			{
				name:     "empty wallet ID",
				playerID: "player-1",
				balance:  validBalance,
				now:      now,
				wantErr:  wallet.ErrInvalidIdentity,
			},
			{
				name:    "empty player ID",
				id:      "wallet-1",
				balance: validBalance,
				now:     now,
				wantErr: wallet.ErrInvalidIdentity,
			},
			{
				name:     "negative balance",
				id:       "wallet-1",
				playerID: "player-1",
				balance:  negativeBalance,
				now:      now,
				wantErr:  wallet.ErrNegativeBalance,
			},
			{
				name:     "uninitialized money",
				id:       "wallet-1",
				playerID: "player-1",
				now:      now,
				wantErr:  money.ErrInvalidMoney,
			},
			{
				name:     "missing timestamp",
				id:       "wallet-1",
				playerID: "player-1",
				balance:  validBalance,
				wantErr:  wallet.ErrInvalidTime,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if _, err := wallet.New(tc.id, tc.playerID, tc.balance, tc.now); !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got: %v", tc.wantErr, err)
				}
			})
		}
	})
}

func TestRehydrate(t *testing.T) {
	createdAt := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)

	t.Run("preserves persisted state", func(t *testing.T) {
		balance := newMoney(t, 7500, "BRL")

		value, err := wallet.Rehydrate(
			"wallet-1", "player-1", balance, 8, createdAt, updatedAt,
		)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if value.ID() != "wallet-1" || value.PlayerID() != "player-1" {
			t.Error("expected identities to be preserved")
		}

		if value.Balance().Amount() != 7500 {
			t.Errorf("expected balance 7500, got %d", value.Balance().Amount())
		}

		if value.Currency() != "BRL" {
			t.Errorf("expected currency BRL, got %s", value.Currency())
		}

		if value.Version() != 8 {
			t.Errorf("expected version 8, got %d", value.Version())
		}

		if !value.CreatedAt().Equal(createdAt) {
			t.Error("expected creation timestamp to be preserved")
		}

		if !value.UpdatedAt().Equal(updatedAt) {
			t.Error("expected update timestamp to be preserved")
		}
	})

	t.Run("rejects invalid persisted state", func(t *testing.T) {
		cases := []struct {
			name      string
			balance   money.Money
			version   int64
			updatedAt time.Time
			wantErr   error
		}{
			{
				name:      "invalid version",
				balance:   newMoney(t, 7500, "BRL"),
				version:   0,
				updatedAt: updatedAt,
				wantErr:   wallet.ErrInvalidVersion,
			},
			{
				name:      "negative balance",
				balance:   newMoney(t, -100, "BRL"),
				version:   8,
				updatedAt: updatedAt,
				wantErr:   wallet.ErrNegativeBalance,
			},
			{
				name:      "uninitialized money",
				version:   8,
				updatedAt: updatedAt,
				wantErr:   money.ErrInvalidMoney,
			},
			{
				name:    "missing update timestamp",
				balance: newMoney(t, 7500, "BRL"),
				version: 8,
				wantErr: wallet.ErrInvalidTime,
			},
			{
				name:      "update before creation",
				balance:   newMoney(t, 7500, "BRL"),
				version:   8,
				updatedAt: createdAt.Add(-time.Minute),
				wantErr:   wallet.ErrInvalidTime,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if _, err := wallet.Rehydrate(
					"wallet-1",
					"player-1",
					tc.balance,
					tc.version,
					createdAt,
					tc.updatedAt,
				); !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got: %v", tc.wantErr, err)
				}
			})
		}
	})
}

func TestCredit(t *testing.T) {
	t.Run("updates balance version and timestamp", func(t *testing.T) {
		createdAt := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
		movementAt := createdAt.Add(time.Minute)

		value, err := wallet.New(
			"wallet-1", "player-1", newMoney(t, 10000, "BRL"), createdAt,
		)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := value.Credit(newMoney(t, 2500, "BRL"), movementAt); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if value.Balance().Amount() != 12500 {
			t.Errorf("expected balance 12500, got %d", value.Balance().Amount())
		}

		if value.Version() != 2 {
			t.Errorf("expected version 2, got %d", value.Version())
		}

		if !value.UpdatedAt().Equal(movementAt) {
			t.Error("expected update timestamp to match movement time")
		}

		if !value.CreatedAt().Equal(createdAt) {
			t.Error("expected creation timestamp to remain unchanged")
		}
	})

	t.Run("zero leaves the complete state unchanged", func(t *testing.T) {
		createdAt := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
		balance := newMoney(t, 10000, "BRL")
		value, err := wallet.New("wallet-1", "player-1", balance, createdAt)

		if err != nil {
			t.Fatal(err)
		}

		if err := value.Credit(newMoney(t, 0, "BRL"), createdAt.Add(time.Minute)); err != nil {
			t.Fatal(err)
		}

		if value.Balance() != balance || value.Version() != 1 ||
			!value.CreatedAt().Equal(createdAt) || !value.UpdatedAt().Equal(createdAt) {
			t.Error("expected balance, version and timestamps to remain unchanged")
		}
	})

	t.Run("rejections preserve the complete state", func(t *testing.T) {
		createdAt := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
		movementAt := createdAt.Add(time.Minute)
		cases := []struct {
			name    string
			balance int64
			amount  money.Money
			now     time.Time
			wantErr error
		}{
			{
				name:    "negative movement",
				balance: 10000,
				amount:  newMoney(t, -100, "BRL"),
				now:     movementAt,
				wantErr: wallet.ErrInvalidMovement,
			},
			{
				name:    "incompatible currency",
				balance: 10000,
				amount:  newMoney(t, 100, "USD"),
				now:     movementAt,
				wantErr: money.ErrCurrencyMismatch,
			},
			{
				name:    "zero with incompatible currency",
				balance: 10000,
				amount:  newMoney(t, 0, "USD"),
				now:     movementAt,
				wantErr: money.ErrCurrencyMismatch,
			},
			{
				name:    "uninitialized money",
				balance: 10000,
				now:     movementAt,
				wantErr: money.ErrInvalidMoney,
			},
			{
				name:    "balance overflow",
				balance: math.MaxInt64,
				amount:  newMoney(t, 1, "BRL"),
				now:     movementAt,
				wantErr: money.ErrOverflow,
			},
			{
				name:    "missing timestamp",
				balance: 10000,
				amount:  newMoney(t, 100, "BRL"),
				wantErr: wallet.ErrInvalidTime,
			},
			{
				name:    "movement before creation",
				balance: 10000,
				amount:  newMoney(t, 100, "BRL"),
				now:     createdAt.Add(-time.Minute),
				wantErr: wallet.ErrInvalidTime,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				balance := newMoney(t, tc.balance, "BRL")
				value, err := wallet.New("wallet-1", "player-1", balance, createdAt)

				if err != nil {
					t.Fatal(err)
				}

				if err := value.Credit(tc.amount, tc.now); !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got: %v", tc.wantErr, err)
				}

				if value.Balance() != balance || value.Version() != 1 ||
					!value.CreatedAt().Equal(createdAt) || !value.UpdatedAt().Equal(createdAt) ||
					value.ID() != "wallet-1" || value.PlayerID() != "player-1" {
					t.Error("expected all wallet state to remain unchanged after rejection")
				}
			})
		}
	})
}

func TestMovementVersionLimit(t *testing.T) {
	createdAt := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	movementAt := createdAt.Add(time.Minute)
	cases := []struct {
		name        string
		apply       func(*wallet.Wallet, money.Money, time.Time) error
		wantBalance int64
	}{
		{
			name:        "credit",
			apply:       (*wallet.Wallet).Credit,
			wantBalance: 10001,
		},
		{
			name:        "debit",
			apply:       (*wallet.Wallet).Debit,
			wantBalance: 9999,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			value, err := wallet.Rehydrate(
				"wallet-1", "player-1", newMoney(t, 10000, "BRL"),
				math.MaxInt64-1, createdAt, createdAt,
			)

			if err != nil {
				t.Fatal(err)
			}

			if err := tc.apply(value, newMoney(t, 1, "BRL"), movementAt); err != nil {
				t.Fatal(err)
			}

			if value.Version() != math.MaxInt64 || value.Balance().Amount() != tc.wantBalance {
				t.Fatal("expected last representable version and correct balance")
			}

			balance := value.Balance()

			if err := tc.apply(value, newMoney(t, 1, "BRL"), movementAt.Add(time.Minute)); !errors.Is(err, wallet.ErrVersionOverflow) {
				t.Fatalf("expected version overflow error, got: %v", err)
			}

			if value.Balance() != balance || value.Version() != math.MaxInt64 ||
				!value.CreatedAt().Equal(createdAt) || !value.UpdatedAt().Equal(movementAt) {
				t.Error("expected state to remain unchanged after version overflow")
			}

			if err := tc.apply(value, newMoney(t, 0, "BRL"), movementAt.Add(2*time.Minute)); err != nil {
				t.Fatal(err)
			}

			if value.Balance() != balance || value.Version() != math.MaxInt64 ||
				!value.UpdatedAt().Equal(movementAt) {
				t.Error("expected zero movement at maximum version to remain a no-op")
			}
		})
	}
}

func TestDebit(t *testing.T) {
	createdAt := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	movementAt := createdAt.Add(time.Minute)

	cases := []struct {
		name        string
		amount      int64
		currency    string
		wantBalance int64
		wantVersion int64
		wantErr     error
	}{
		{
			name:   "debits balance",
			amount: 2500, currency: "BRL",
			wantBalance: 7500, wantVersion: 2,
		},
		{
			name:   "allows spending the full balance",
			amount: 10000, currency: "BRL",
			wantBalance: 0, wantVersion: 2,
		},
		{
			name:   "rejects insufficient funds",
			amount: 10001, currency: "BRL",
			wantBalance: 10000, wantVersion: 1,
			wantErr: wallet.ErrInsufficientFunds,
		},
		{
			name:   "zero leaves state unchanged",
			amount: 0, currency: "BRL",
			wantBalance: 10000, wantVersion: 1,
		},
		{
			name:   "rejects negative movement",
			amount: -100, currency: "BRL",
			wantBalance: 10000, wantVersion: 1,
			wantErr: wallet.ErrInvalidMovement,
		},
		{
			name:   "rejects incompatible currency",
			amount: 100, currency: "USD",
			wantBalance: 10000, wantVersion: 1,
			wantErr: money.ErrCurrencyMismatch,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			value, err := wallet.New(
				"wallet-1", "player-1", newMoney(t, 10000, "BRL"), createdAt,
			)

			if err != nil {
				t.Fatalf("unexpected creation error: %v", err)
			}

			if err := value.Debit(newMoney(t, tc.amount, tc.currency), movementAt); !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected error %v, got: %v", tc.wantErr, err)
			}

			if value.Balance().Amount() != tc.wantBalance {
				t.Errorf("expected balance %d, got %d",
					tc.wantBalance, value.Balance().Amount())
			}

			if value.Version() != tc.wantVersion {
				t.Errorf("expected version %d, got %d",
					tc.wantVersion, value.Version())
			}

			wantUpdatedAt := createdAt

			if tc.wantVersion == 2 {
				wantUpdatedAt = movementAt
			}

			if !value.UpdatedAt().Equal(wantUpdatedAt) {
				t.Errorf("expected updatedAt %v, got %v",
					wantUpdatedAt, value.UpdatedAt())
			}
		})
	}
}
