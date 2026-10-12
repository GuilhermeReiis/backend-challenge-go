package wager_test

import (
	"errors"
	"testing"
	"time"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/wager"
)

func validSnapshot(t *testing.T) wager.Snapshot {
	t.Helper()

	input := validInput(t)
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

	return wager.Snapshot{
		ID:                    input.ID,
		Origin:                wager.ExternalOrigin,
		ProviderID:            input.ProviderID,
		ExternalTransactionID: input.ExternalTransactionID,
		IdempotencyKey:        input.IdempotencyKey,
		PayloadHash:           input.PayloadHash,
		WalletID:              input.WalletID,
		PlayerID:              input.PlayerID,
		RoundID:               input.RoundID,
		GameID:                input.GameID,
		Kind:                  input.Kind,
		Money:                 input.Money,
		Status:                wager.Pending,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
}

func TestRehydrate(t *testing.T) {
	t.Run("preserves processed state and original balance", func(t *testing.T) {
		state := validSnapshot(t)
		state.Status = wager.Processed
		state.UpdatedAt = state.CreatedAt.Add(time.Minute)
		state.ProcessedAt = state.UpdatedAt

		balance, err := money.Parse("75.00", "BRL")

		if err != nil {
			t.Fatal(err)
		}

		state.ResultBalance = balance

		value, err := wager.Rehydrate(state)

		if err != nil {
			t.Fatal(err)
		}

		if value.ID() != state.ID ||
			value.WalletID() != state.WalletID ||
			value.PlayerID() != state.PlayerID ||
			value.ProviderID() != state.ProviderID ||
			value.ExternalTransactionID() != state.ExternalTransactionID ||
			value.IdempotencyKey() != state.IdempotencyKey ||
			value.PayloadHash() != state.PayloadHash ||
			value.RoundID() != state.RoundID ||
			value.GameID() != state.GameID {
			t.Error("expected persisted identities and metadata")
		}

		if value.Origin() != state.Origin ||
			value.Kind() != state.Kind ||
			value.Status() != wager.Processed {
			t.Error("expected persisted origin kind and status")
		}

		if value.Money() != state.Money {
			t.Error("expected transaction amount to be preserved")
		}

		result, available := value.ResultBalance()

		if !available || result != balance {
			t.Error("expected original processing balance")
		}

		if !value.CreatedAt().Equal(state.CreatedAt) ||
			!value.UpdatedAt().Equal(state.UpdatedAt) ||
			!value.ProcessedAt().Equal(state.ProcessedAt) {
			t.Error("expected persisted timestamps")
		}

		if err := value.Process(balance, "", state.UpdatedAt.Add(time.Minute)); !errors.Is(err, wager.ErrTerminalState) {
			t.Fatalf("expected terminal state error, got: %v", err)
		}
	})

	t.Run("preserves pending rejected and failed states", func(t *testing.T) {
		cases := []struct {
			name        string
			status      wager.Status
			failureCode string
		}{
			{
				name:   "pending",
				status: wager.Pending,
			},
			{
				name:        "rejected",
				status:      wager.Rejected,
				failureCode: "INSUFFICIENT_FUNDS",
			},
			{
				name:        "failed",
				status:      wager.Failed,
				failureCode: "PERMANENT_INFRASTRUCTURE_FAILURE",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				state := validSnapshot(t)
				state.Status = tc.status
				state.FailureCode = tc.failureCode

				value, err := wager.Rehydrate(state)

				if err != nil {
					t.Fatal(err)
				}

				if value.Status() != tc.status {
					t.Errorf("expected status %s, got %s",
						tc.status, value.Status())
				}

				if value.FailureCode() != tc.failureCode {
					t.Errorf("expected failure code %q, got %q",
						tc.failureCode, value.FailureCode())
				}

				_, available := value.ResultBalance()

				if available {
					t.Error("expected no successful processing balance")
				}

				if !value.UpdatedAt().Equal(state.UpdatedAt) {
					t.Error("expected update timestamp to be preserved")
				}
			})
		}
	})

	t.Run("rejects inconsistent persisted state", func(t *testing.T) {
		cases := []struct {
			name    string
			change  func(*wager.Snapshot)
			wantErr error
		}{
			{
				name: "missing identity",
				change: func(state *wager.Snapshot) {
					state.ID = ""
				},
				wantErr: wager.ErrInvalidIdentity,
			},
			{
				name: "invalid timestamp",
				change: func(state *wager.Snapshot) {
					state.UpdatedAt = state.CreatedAt.Add(-time.Minute)
				},
				wantErr: wager.ErrInvalidTime,
			},
			{
				name: "rejected without failure code",
				change: func(state *wager.Snapshot) {
					state.Status = wager.Rejected
				},
				wantErr: wager.ErrInvalidFailureCode,
			},
			{
				name: "processed without result balance",
				change: func(state *wager.Snapshot) {
					state.Status = wager.Processed
					state.ProcessedAt = state.UpdatedAt
				},
				wantErr: money.ErrInvalidMoney,
			},
			{
				name: "pending reference without reference",
				change: func(state *wager.Snapshot) {
					state.Status = wager.PendingReference
				},
				wantErr: wager.ErrInvalidReference,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				state := validSnapshot(t)
				tc.change(&state)

				value, err := wager.Rehydrate(state)

				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got: %v", tc.wantErr, err)
				}

				if value != nil {
					t.Error("expected no transaction on validation failure")
				}
			})
		}
	})

	t.Run("restores internal opening without external metadata", func(t *testing.T) {
		state := validSnapshot(t)
		state.Origin = wager.InternalOrigin
		state.Kind = wager.Opening
		state.ProviderID = ""
		state.ExternalTransactionID = ""
		state.IdempotencyKey = ""
		state.PayloadHash = ""
		state.RoundID = ""
		state.GameID = ""
		state.Status = wager.Processed
		state.ProcessedAt = state.UpdatedAt
		state.ResultBalance = state.Money

		value, err := wager.Rehydrate(state)

		if err != nil {
			t.Fatal(err)
		}

		if value.Origin() != wager.InternalOrigin ||
			value.Kind() != wager.Opening ||
			value.Status() != wager.Processed {
			t.Error("expected processed internal opening")
		}

		if value.ProviderID() != "" ||
			value.ExternalTransactionID() != "" ||
			value.IdempotencyKey() != "" ||
			value.PayloadHash() != "" ||
			value.RoundID() != "" ||
			value.GameID() != "" {
			t.Error("expected no external metadata")
		}

		result, available := value.ResultBalance()

		if !available || result != state.ResultBalance {
			t.Error("expected original opening balance")
		}
	})

	t.Run("preserves pending and resolved references", func(t *testing.T) {
		cases := []struct {
			name   string
			status wager.Status
		}{
			{
				name:   "pending reference",
				status: wager.PendingReference,
			},
			{
				name:   "processed reference",
				status: wager.Processed,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				state := validSnapshot(t)
				state.Kind = wager.Refund
				state.ReferenceExternalTransactionID = "external-bet-1"
				state.Status = tc.status

				if tc.status == wager.Processed {
					state.ReferenceTransactionID = "internal-bet-1"
					state.ProcessedAt = state.UpdatedAt
					state.ResultBalance = state.Money
				}

				value, err := wager.Rehydrate(state)

				if err != nil {
					t.Fatal(err)
				}

				if value.Status() != tc.status {
					t.Errorf("expected status %s, got %s",
						tc.status, value.Status())
				}

				if value.ReferenceExternalTransactionID() != state.ReferenceExternalTransactionID {
					t.Error("expected external reference to be preserved")
				}

				if value.ReferenceTransactionID() != state.ReferenceTransactionID {
					t.Error("expected internal reference to be preserved")
				}

				if !value.UpdatedAt().Equal(state.UpdatedAt) {
					t.Error("expected timestamp to remain unchanged")
				}
			})
		}
	})

	t.Run("rejects invalid processing balances", func(t *testing.T) {
		cases := []struct {
			name     string
			amount   int64
			currency string
			wantErr  error
		}{
			{
				name:     "negative balance",
				amount:   -100,
				currency: "BRL",
				wantErr:  wager.ErrInvalidAmount,
			},
			{
				name:     "incompatible currency",
				amount:   7500,
				currency: "USD",
				wantErr:  money.ErrCurrencyMismatch,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				state := validSnapshot(t)
				state.Status = wager.Processed
				state.ProcessedAt = state.UpdatedAt

				balance, err := money.New(tc.amount, tc.currency)

				if err != nil {
					t.Fatal(err)
				}

				state.ResultBalance = balance

				value, err := wager.Rehydrate(state)

				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got: %v", tc.wantErr, err)
				}

				if value != nil {
					t.Error("expected no transaction on validation failure")
				}
			})
		}
	})

	t.Run("keeps restored rejected and failed states terminal", func(t *testing.T) {
		for _, status := range []wager.Status{wager.Rejected, wager.Failed} {
			t.Run(string(status), func(t *testing.T) {
				state := validSnapshot(t)
				state.Status = status
				state.FailureCode = "ORIGINAL_FAILURE"

				value, err := wager.Rehydrate(state)

				if err != nil {
					t.Fatal(err)
				}

				if err := value.Reject("OTHER_FAILURE", state.UpdatedAt.Add(time.Minute)); !errors.Is(err, wager.ErrTerminalState) {
					t.Fatalf("expected terminal state error, got: %v", err)
				}

				if value.Status() != status ||
					value.FailureCode() != state.FailureCode ||
					!value.UpdatedAt().Equal(state.UpdatedAt) {
					t.Error("expected terminal state to remain unchanged")
				}
			})
		}
	})
}
