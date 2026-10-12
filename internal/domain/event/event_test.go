package event_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/event"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/ledger"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/wager"
)

func newPendingTransaction(
	t *testing.T,
	kind wager.Kind,
	reference string,
	now time.Time,
) *wager.Transaction {
	t.Helper()

	amountText := "25.00"

	if kind == wager.Loss {
		amountText = "0.00"
	}

	amount, err := money.Parse(amountText, "BRL")

	if err != nil {
		t.Fatal(err)
	}

	value, err := wager.NewExternal(wager.ExternalInput{
		ID:                             "transaction-1",
		ProviderID:                     "provider-a",
		ExternalTransactionID:          "external-1",
		IdempotencyKey:                 "key-1",
		PayloadHash:                    strings.Repeat("a", 64),
		WalletID:                       "wallet-1",
		PlayerID:                       "player-1",
		RoundID:                        "round-1",
		GameID:                         "game-1",
		Kind:                           kind,
		Money:                          amount,
		ReferenceExternalTransactionID: reference,
	}, now)

	if err != nil {
		t.Fatal(err)
	}

	return value
}

func TestEventMetadata(t *testing.T) {
	now := time.Date(2026, 10, 12, 9, 0, 0, 0, time.FixedZone("UTC-3", -3*60*60))
	processed := newPendingTransaction(t, wager.Bet, "", now)
	amount := processed.Money()

	if err := processed.Process(amount, "", now); err != nil {
		t.Fatal(err)
	}

	rejected := newPendingTransaction(t, wager.Bet, "", now)

	if err := rejected.Reject("INSUFFICIENT_FUNDS", now); err != nil {
		t.Fatal(err)
	}

	pending := newPendingTransaction(t, wager.Refund, "bet-1", now)

	if err := pending.AwaitReference(now); err != nil {
		t.Fatal(err)
	}

	zero, err := money.Zero("BRL")

	if err != nil {
		t.Fatal(err)
	}

	entry, err := ledger.New("entry-1", "wallet-1", "transaction-1", ledger.Credit, amount, zero, amount, now)

	if err != nil {
		t.Fatal(err)
	}

	constructors := []struct {
		name   string
		create func(event.Metadata) (json.Marshaler, error)
	}{
		{
			name: "WagerTransactionProcessed",
			create: func(metadata event.Metadata) (json.Marshaler, error) {
				return event.NewProcessed(metadata, processed)
			},
		},
		{
			name: "WagerTransactionRejected",
			create: func(metadata event.Metadata) (json.Marshaler, error) {
				return event.NewRejected(metadata, rejected)
			},
		},
		{
			name: "WagerTransactionPendingReference",
			create: func(metadata event.Metadata) (json.Marshaler, error) {
				return event.NewPendingReference(metadata, pending)
			},
		},
		{
			name: "WalletBalanceChanged",
			create: func(metadata event.Metadata) (json.Marshaler, error) {
				return event.NewBalanceChanged(metadata, entry, 1)
			},
		},
	}
	cases := []struct {
		name   string
		change func(*event.Metadata)
	}{
		{name: "missing event ID", change: func(m *event.Metadata) { m.EventID = "" }},
		{name: "blank event ID", change: func(m *event.Metadata) { m.EventID = " \t" }},
		{name: "missing correlation ID", change: func(m *event.Metadata) { m.CorrelationID = "" }},
		{name: "blank correlation ID", change: func(m *event.Metadata) { m.CorrelationID = " \t" }},
		{name: "missing timestamp", change: func(m *event.Metadata) { m.OccurredAt = time.Time{} }},
		{name: "blank causation ID", change: func(m *event.Metadata) { m.CausationID = " \t" }},
	}
	metadata := event.Metadata{EventID: "event-1", CorrelationID: "correlation-1", OccurredAt: now}

	for _, constructor := range constructors {
		t.Run(constructor.name, func(t *testing.T) {
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					invalid := metadata
					tc.change(&invalid)

					if _, err := constructor.create(invalid); !errors.Is(err, event.ErrInvalidMetadata) {
						t.Fatalf("expected invalid metadata error, got: %v", err)
					}
				})
			}

			for _, causationID := range []string{"", "message-1"} {
				t.Run("serializes metadata with causation "+causationID, func(t *testing.T) {
					input := metadata
					input.CausationID = causationID
					value, err := constructor.create(input)

					if err != nil {
						t.Fatal(err)
					}

					input.EventID = "changed-after-creation"
					result, err := json.Marshal(value)

					if err != nil {
						t.Fatal(err)
					}

					var decoded struct {
						EventID       string  `json:"eventId"`
						EventType     string  `json:"eventType"`
						AggregateID   string  `json:"aggregateId"`
						CorrelationID string  `json:"correlationId"`
						CausationID   *string `json:"causationId"`
						OccurredAt    string  `json:"occurredAt"`
						Version       int     `json:"version"`
					}

					if err := json.Unmarshal(result, &decoded); err != nil {
						t.Fatal(err)
					}

					if decoded.EventID != metadata.EventID || decoded.EventType != constructor.name ||
						decoded.AggregateID != "wallet-1" || decoded.CorrelationID != metadata.CorrelationID ||
						decoded.OccurredAt != "2026-10-12T12:00:00Z" || decoded.Version != 1 {
						t.Errorf("unexpected envelope: %s", result)
					}

					if causationID == "" {
						if decoded.CausationID != nil {
							t.Error("expected optional causation ID to be omitted")
						}
					} else if decoded.CausationID == nil || *decoded.CausationID != causationID {
						t.Error("expected causation ID to be preserved")
					}
				})
			}
		})
	}
}

func TestUninitializedEvents(t *testing.T) {
	metadata := event.Metadata{
		EventID: "event-1", CorrelationID: "correlation-1",
		OccurredAt: time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC),
	}
	cases := []struct {
		name        string
		transaction *wager.Transaction
	}{
		{name: "nil transaction"},
		{name: "zero transaction", transaction: &wager.Transaction{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := event.NewProcessed(metadata, tc.transaction); !errors.Is(err, event.ErrInvalidData) {
				t.Errorf("expected processed event to reject invalid data, got: %v", err)
			}

			if _, err := event.NewRejected(metadata, tc.transaction); !errors.Is(err, event.ErrInvalidData) {
				t.Errorf("expected rejected event to reject invalid data, got: %v", err)
			}

			if _, err := event.NewPendingReference(metadata, tc.transaction); !errors.Is(err, event.ErrInvalidData) {
				t.Errorf("expected pending reference event to reject invalid data, got: %v", err)
			}
		})
	}

	envelopes := []struct {
		name  string
		value json.Marshaler
	}{
		{name: "processed", value: event.Envelope[event.ProcessedData]{}},
		{name: "rejected", value: event.Envelope[event.RejectedData]{}},
		{name: "pending reference", value: event.Envelope[event.PendingReferenceData]{}},
		{name: "balance changed", value: event.Envelope[event.BalanceChangedData]{}},
	}

	for _, envelope := range envelopes {
		t.Run("zero envelope "+envelope.name, func(t *testing.T) {
			if _, err := json.Marshal(envelope.value); !errors.Is(err, event.ErrInvalidMetadata) {
				t.Fatalf("expected invalid metadata error, got: %v", err)
			}
		})
	}
}

func TestNewRejected(t *testing.T) {
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	metadata := event.Metadata{
		EventID:       "event-1",
		CorrelationID: "correlation-1",
		OccurredAt:    now,
	}

	t.Run("includes the rejection reason", func(t *testing.T) {
		transaction := newPendingTransaction(t, wager.Bet, "", now)

		if err := transaction.Reject("INSUFFICIENT_FUNDS", now); err != nil {
			t.Fatal(err)
		}

		value, err := event.NewRejected(metadata, transaction)

		if err != nil {
			t.Fatal(err)
		}

		result, err := json.Marshal(value)

		if err != nil {
			t.Fatal(err)
		}

		var decoded struct {
			EventType string `json:"eventType"`
			Data      struct {
				TransactionID string `json:"transactionId"`
				FailureCode   string `json:"failureCode"`
			} `json:"data"`
		}

		if err := json.Unmarshal(result, &decoded); err != nil {
			t.Fatal(err)
		}

		if decoded.EventType != "WagerTransactionRejected" ||
			decoded.Data.TransactionID != transaction.ID() ||
			decoded.Data.FailureCode != "INSUFFICIENT_FUNDS" {
			t.Error("expected rejection event with original failure code")
		}
	})

	t.Run("rejects a pending transaction", func(t *testing.T) {
		transaction := newPendingTransaction(t, wager.Bet, "", now)

		if _, err := event.NewRejected(metadata, transaction); !errors.Is(err, event.ErrInvalidData) {
			t.Fatalf("expected invalid data error, got: %v", err)
		}
	})
}

func TestNewPendingReference(t *testing.T) {
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	metadata := event.Metadata{
		EventID:       "event-1",
		CorrelationID: "correlation-1",
		OccurredAt:    now,
	}

	t.Run("includes the awaited reference", func(t *testing.T) {
		transaction := newPendingTransaction(t, wager.Refund, "bet-1", now)

		if err := transaction.AwaitReference(now); err != nil {
			t.Fatal(err)
		}

		value, err := event.NewPendingReference(metadata, transaction)

		if err != nil {
			t.Fatal(err)
		}

		result, err := json.Marshal(value)

		if err != nil {
			t.Fatal(err)
		}

		var decoded struct {
			EventType string `json:"eventType"`
			Data      struct {
				Reference string `json:"referenceExternalTransactionId"`
			} `json:"data"`
		}

		if err := json.Unmarshal(result, &decoded); err != nil {
			t.Fatal(err)
		}

		if decoded.EventType != "WagerTransactionPendingReference" ||
			decoded.Data.Reference != "bet-1" {
			t.Error("expected pending reference event")
		}
	})

	t.Run("rejects an ordinary pending transaction", func(t *testing.T) {
		transaction := newPendingTransaction(t, wager.Refund, "bet-1", now)

		if _, err := event.NewPendingReference(metadata, transaction); !errors.Is(err, event.ErrInvalidData) {
			t.Fatalf("expected invalid data error, got: %v", err)
		}
	})
}

func TestNewBalanceChanged(t *testing.T) {
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	metadata := event.Metadata{
		EventID:       "event-1",
		CorrelationID: "correlation-1",
		OccurredAt:    now,
	}

	t.Run("includes the ledger movement and wallet version", func(t *testing.T) {
		before, err := money.Parse("100.00", "BRL")

		if err != nil {
			t.Fatal(err)
		}

		amount, err := money.Parse("25.00", "BRL")

		if err != nil {
			t.Fatal(err)
		}

		after, err := before.Sub(amount)

		if err != nil {
			t.Fatal(err)
		}

		entry, err := ledger.New(
			"entry-1", "wallet-1", "transaction-1",
			ledger.Debit, amount, before, after, now,
		)

		if err != nil {
			t.Fatal(err)
		}

		value, err := event.NewBalanceChanged(metadata, entry, 2)

		if err != nil {
			t.Fatal(err)
		}

		result, err := json.Marshal(value)

		if err != nil {
			t.Fatal(err)
		}

		expected := `{"eventId":"event-1","eventType":"WalletBalanceChanged",` +
			`"aggregateId":"wallet-1","correlationId":"correlation-1",` +
			`"occurredAt":"2026-10-12T12:00:00Z","version":1,` +
			`"data":{"walletId":"wallet-1","transactionId":"transaction-1",` +
			`"direction":"DEBIT","money":{"amount":"25.00","currency":"BRL"},` +
			`"balanceBefore":{"amount":"100.00","currency":"BRL"},` +
			`"balanceAfter":{"amount":"75.00","currency":"BRL"},"walletVersion":2}}`

		if string(result) != expected {
			t.Errorf("expected %s, got %s", expected, result)
		}

		if _, err := event.NewBalanceChanged(metadata, entry, 0); !errors.Is(err, event.ErrInvalidData) {
			t.Fatalf("expected invalid version rejection, got: %v", err)
		}
	})

	t.Run("rejects an uninitialized ledger entry", func(t *testing.T) {
		if _, err := event.NewBalanceChanged(metadata, ledger.Entry{}, 1); !errors.Is(err, event.ErrInvalidData) {
			t.Fatalf("expected invalid data error, got: %v", err)
		}
	})
}
