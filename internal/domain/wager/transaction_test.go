package wager_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/wager"
)

func validInput(t *testing.T) wager.ExternalInput {
	t.Helper()

	amount, err := money.Parse("25.00", "BRL")

	if err != nil {
		t.Fatalf("unexpected money error: %v", err)
	}

	return wager.ExternalInput{
		ID:                    "transaction-1",
		ProviderID:            "provider-a",
		ExternalTransactionID: "external-1",
		IdempotencyKey:        "provider-a:external-1",
		PayloadHash:           strings.Repeat("a", 64),
		WalletID:              "wallet-1",
		PlayerID:              "player-1",
		RoundID:               "round-1",
		GameID:                "game-1",
		Kind:                  wager.Bet,
		Money:                 amount,
	}
}

func TestTransitionTimestamps(t *testing.T) {
	createdAt := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(2 * time.Hour)

	transitions := []struct {
		name   string
		apply  func(*wager.Transaction, time.Time) error
		status wager.Status
	}{
		{
			name:   "await reference",
			apply:  (*wager.Transaction).AwaitReference,
			status: wager.PendingReference,
		},
		{
			name: "process",
			apply: func(value *wager.Transaction, now time.Time) error {
				balance, err := money.Parse("100.00", "BRL")

				if err != nil {
					return err
				}

				return value.Process(balance, "internal-bet-1", now)
			},
			status: wager.Processed,
		},
		{
			name: "reject",
			apply: func(value *wager.Transaction, now time.Time) error {
				return value.Reject("REFERENCE_NOT_FOUND", now)
			},
			status: wager.Rejected,
		},
		{
			name: "fail",
			apply: func(value *wager.Transaction, now time.Time) error {
				return value.Fail("PERMANENT_INFRASTRUCTURE_FAILURE", now)
			},
			status: wager.Failed,
		},
	}

	for _, transition := range transitions {
		t.Run(transition.name, func(t *testing.T) {
			cases := []struct {
				name    string
				now     time.Time
				wantErr error
			}{
				{
					name:    "rejects time before the last update",
					now:     createdAt.Add(time.Hour),
					wantErr: wager.ErrInvalidTime,
				},
				{
					name: "accepts time equal to the last update",
					now:  updatedAt,
				},
			}

			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					input := validInput(t)
					input.Kind = wager.Refund
					input.ReferenceExternalTransactionID = "bet-1"
					value := newTransaction(t, input, createdAt)

					if err := value.AwaitReference(updatedAt); err != nil {
						t.Fatal(err)
					}

					before := *value

					if err := transition.apply(value, tc.now); !errors.Is(err, tc.wantErr) {
						t.Fatalf("expected error %v, got: %v", tc.wantErr, err)
					}

					if tc.wantErr != nil {
						if *value != before {
							t.Error("expected rejected transition to preserve the complete state")
						}

						return
					}

					if value.Status() != transition.status ||
						!value.UpdatedAt().Equal(updatedAt) {
						t.Error("expected transition to succeed with the accepted timestamp")
					}
				})
			}
		})
	}
}

func TestNewExternal(t *testing.T) {
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

	t.Run("creates valid operations in pending state", func(t *testing.T) {
		cases := []struct {
			name      string
			kind      wager.Kind
			amount    int64
			reference string
		}{
			{name: "bet", kind: wager.Bet, amount: 2500},
			{name: "win", kind: wager.Win, amount: 5000},
			{
				name: "win with reference",
				kind: wager.Win, amount: 5000, reference: "bet-1",
			},
			{name: "loss", kind: wager.Loss, amount: 0},
			{
				name: "refund",
				kind: wager.Refund, amount: 2500, reference: "bet-1",
			},
			{
				name: "rollback",
				kind: wager.Rollback, amount: 2500, reference: "bet-1",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				input := validInput(t)
				input.Kind = tc.kind
				input.ReferenceExternalTransactionID = tc.reference

				amount, err := money.New(tc.amount, "BRL")

				if err != nil {
					t.Fatalf("unexpected money error: %v", err)
				}

				input.Money = amount

				value, err := wager.NewExternal(input, now)

				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if value.ID() != input.ID {
					t.Error("expected transaction ID to be preserved")
				}

				if value.Status() != wager.Pending {
					t.Errorf("expected PENDING, got %s", value.Status())
				}

				if value.Kind() != tc.kind {
					t.Errorf("expected kind %s, got %s", tc.kind, value.Kind())
				}

				if value.Money().Amount() != tc.amount {
					t.Errorf("expected amount %d, got %d",
						tc.amount, value.Money().Amount())
				}

				if !value.CreatedAt().Equal(now) ||
					!value.UpdatedAt().Equal(now) {
					t.Error("expected timestamps to match creation time")
				}

				if value.Origin() != wager.ExternalOrigin {
					t.Errorf("expected EXTERNAL, got %s", value.Origin())
				}
			})
		}
	})

	t.Run("rejects invalid operations", func(t *testing.T) {
		cases := []struct {
			name      string
			kind      wager.Kind
			amount    int64
			reference string
			wantErr   error
		}{
			{
				name: "external opening",
				kind: wager.Opening, amount: 2500,
				wantErr: wager.ErrInvalidKind,
			},
			{
				name: "unknown kind",
				kind: wager.Kind("UNKNOWN"), amount: 2500,
				wantErr: wager.ErrInvalidKind,
			},
			{
				name: "zero bet",
				kind: wager.Bet, amount: 0,
				wantErr: wager.ErrInvalidAmount,
			},
			{
				name: "negative win",
				kind: wager.Win, amount: -100,
				wantErr: wager.ErrInvalidAmount,
			},
			{
				name: "positive loss",
				kind: wager.Loss, amount: 100,
				wantErr: wager.ErrInvalidAmount,
			},
			{
				name: "refund without reference",
				kind: wager.Refund, amount: 2500,
				wantErr: wager.ErrInvalidReference,
			},
			{
				name: "rollback without reference",
				kind: wager.Rollback, amount: 2500,
				wantErr: wager.ErrInvalidReference,
			},
			{
				name: "bet with reference",
				kind: wager.Bet, amount: 2500, reference: "bet-1",
				wantErr: wager.ErrInvalidReference,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				input := validInput(t)
				input.Kind = tc.kind
				input.ReferenceExternalTransactionID = tc.reference

				amount, err := money.New(tc.amount, "BRL")

				if err != nil {
					t.Fatalf("unexpected money error: %v", err)
				}

				input.Money = amount

				if _, err := wager.NewExternal(input, now); !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got: %v", tc.wantErr, err)
				}
			})
		}
	})
}

func TestProcess(t *testing.T) {
	t.Run("stores original processing result", func(t *testing.T) {
		createdAt := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
		processedAt := createdAt.Add(time.Minute)

		value := newTransaction(t, validInput(t), createdAt)

		balance, err := money.Parse("75.00", "BRL")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := value.Process(balance, "", processedAt); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if value.Status() != wager.Processed {
			t.Errorf("expected PROCESSED, got %s", value.Status())
		}

		result, available := value.ResultBalance()

		if !available || result.Amount() != 7500 {
			t.Error("expected original processing balance")
		}

		if !value.ProcessedAt().Equal(processedAt) {
			t.Error("expected processing timestamp")
		}
	})
}

func TestAwaitReference(t *testing.T) {
	t.Run("waits and later processes with resolved reference", func(t *testing.T) {
		createdAt := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

		input := validInput(t)
		input.Kind = wager.Refund
		input.ReferenceExternalTransactionID = "bet-1"

		value := newTransaction(t, input, createdAt)

		if err := value.AwaitReference(createdAt.Add(time.Minute)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if value.Status() != wager.PendingReference {
			t.Errorf("expected PENDING_REFERENCE, got %s", value.Status())
		}

		balance, err := money.Parse("100.00", "BRL")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := value.Process(balance, "internal-bet-1", createdAt.Add(2*time.Minute)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if value.ReferenceTransactionID() != "internal-bet-1" {
			t.Error("expected resolved internal reference")
		}
	})
}

func TestTerminalStates(t *testing.T) {
	createdAt := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name     string
		finalize func(*wager.Transaction, time.Time) error
		want     wager.Status
	}{
		{
			name: "rejected",
			finalize: func(value *wager.Transaction, now time.Time) error {
				return value.Reject("INSUFFICIENT_FUNDS", now)
			},
			want: wager.Rejected,
		},
		{
			name: "failed",
			finalize: func(value *wager.Transaction, now time.Time) error {
				return value.Fail("PERMANENT_INFRASTRUCTURE_FAILURE", now)
			},
			want: wager.Failed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			value := newTransaction(t, validInput(t), createdAt)
			finishedAt := createdAt.Add(time.Minute)

			if err := tc.finalize(value, finishedAt); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if err := value.Reject("OTHER_REASON", finishedAt.Add(time.Minute)); !errors.Is(err, wager.ErrTerminalState) {
				t.Fatalf("expected terminal state error, got: %v", err)
			}

			if value.Status() != tc.want {
				t.Error("expected terminal status to remain unchanged")
			}

			if !value.UpdatedAt().Equal(finishedAt) {
				t.Error("expected timestamp to remain unchanged")
			}
		})
	}
}

func TestNewOpening(t *testing.T) {
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

	t.Run("creates an internal opening without external metadata", func(t *testing.T) {
		amount, err := money.Parse("100.00", "BRL")

		if err != nil {
			t.Fatalf("unexpected money error: %v", err)
		}

		value, err := wager.NewOpening(wager.OpeningInput{
			ID:       "opening-1",
			WalletID: "wallet-1",
			PlayerID: "player-1",
			Money:    amount,
		}, now)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if value.Origin() != wager.InternalOrigin {
			t.Errorf("expected INTERNAL, got %s", value.Origin())
		}

		if value.Kind() != wager.Opening {
			t.Errorf("expected OPENING, got %s", value.Kind())
		}

		if value.Status() != wager.Pending {
			t.Errorf("expected PENDING, got %s", value.Status())
		}

		if value.WalletID() != "wallet-1" || value.PlayerID() != "player-1" {
			t.Error("expected wallet and player identities")
		}

		if value.ProviderID() != "" ||
			value.ExternalTransactionID() != "" ||
			value.IdempotencyKey() != "" ||
			value.PayloadHash() != "" ||
			value.RoundID() != "" ||
			value.GameID() != "" {
			t.Error("expected no external metadata")
		}

		if err := value.Process(amount, "", now); err != nil {
			t.Fatalf("unexpected processing error: %v", err)
		}

		if value.Status() != wager.Processed {
			t.Errorf("expected PROCESSED, got %s", value.Status())
		}

		result, available := value.ResultBalance()

		if !available || result.Amount() != 10000 {
			t.Error("expected opening balance of 10000")
		}
	})

	t.Run("rejects invalid opening amounts", func(t *testing.T) {
		cases := []struct {
			name   string
			amount int64
		}{
			{name: "zero", amount: 0},
			{name: "negative", amount: -100},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				amount, err := money.New(tc.amount, "BRL")

				if err != nil {
					t.Fatalf("unexpected money error: %v", err)
				}

				if _, err := wager.NewOpening(wager.OpeningInput{
					ID:       "opening-1",
					WalletID: "wallet-1",
					PlayerID: "player-1",
					Money:    amount,
				}, now); !errors.Is(err, wager.ErrInvalidAmount) {
					t.Fatalf("expected invalid amount error, got: %v", err)
				}
			})
		}
	})
}

func newTransaction(t *testing.T, input wager.ExternalInput, now time.Time) *wager.Transaction {
	t.Helper()

	value, err := wager.NewExternal(input, now)

	if err != nil {
		t.Fatalf("failed to create transaction: %v", err)
	}

	return value
}
