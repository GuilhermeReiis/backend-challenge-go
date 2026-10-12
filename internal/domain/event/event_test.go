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

	amount, err := money.Parse("25.00", "BRL")
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

func TestNewRejected(t *testing.T) {
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	metadata := event.Metadata{
		EventID:       "event-1",
		CorrelationID: "correlation-1",
		OccurredAt:    now,
	}

	t.Run("includes the rejection reason", func(t *testing.T) {
		transaction := newPendingTransaction(t, wager.Bet, "", now)

		err := transaction.Reject("INSUFFICIENT_FUNDS", now)
		if err != nil {
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

		err = json.Unmarshal(result, &decoded)
		if err != nil {
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

		_, err := event.NewRejected(metadata, transaction)
		if !errors.Is(err, event.ErrInvalidData) {
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

		err := transaction.AwaitReference(now)
		if err != nil {
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

		err = json.Unmarshal(result, &decoded)
		if err != nil {
			t.Fatal(err)
		}

		if decoded.EventType != "WagerTransactionPendingReference" ||
			decoded.Data.Reference != "bet-1" {
			t.Error("expected pending reference event")
		}
	})

	t.Run("rejects an ordinary pending transaction", func(t *testing.T) {
		transaction := newPendingTransaction(t, wager.Refund, "bet-1", now)

		_, err := event.NewPendingReference(metadata, transaction)
		if !errors.Is(err, event.ErrInvalidData) {
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

		_, err = event.NewBalanceChanged(metadata, entry, 0)
		if !errors.Is(err, event.ErrInvalidData) {
			t.Fatalf("expected invalid version rejection, got: %v", err)
		}
	})

	t.Run("rejects an uninitialized ledger entry", func(t *testing.T) {
		_, err := event.NewBalanceChanged(metadata, ledger.Entry{}, 1)
		if !errors.Is(err, event.ErrInvalidData) {
			t.Fatalf("expected invalid data error, got: %v", err)
		}
	})
}
