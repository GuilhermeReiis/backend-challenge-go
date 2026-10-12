package event_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/event"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/wager"
)

func TestNewProcessed(t *testing.T) {
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)

	metadata := event.Metadata{
		EventID:       "event-1",
		CorrelationID: "correlation-1",
		OccurredAt:    now,
	}

	t.Run("serializes a processed opening with exact money", func(t *testing.T) {
		amount, err := money.Parse("100.00", "BRL")
		if err != nil {
			t.Fatal(err)
		}

		transaction, err := wager.NewOpening(wager.OpeningInput{
			ID:       "opening-1",
			WalletID: "wallet-1",
			PlayerID: "player-1",
			Money:    amount,
		}, now)
		if err != nil {
			t.Fatal(err)
		}

		err = transaction.Process(amount, "", now)
		if err != nil {
			t.Fatal(err)
		}

		value, err := event.NewProcessed(metadata, transaction)
		if err != nil {
			t.Fatal(err)
		}

		result, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}

		expected := `{"eventId":"event-1","eventType":"WagerTransactionProcessed",` +
			`"aggregateId":"wallet-1","correlationId":"correlation-1",` +
			`"occurredAt":"2026-10-12T12:00:00Z","version":1,` +
			`"data":{"transactionId":"opening-1","walletId":"wallet-1",` +
			`"playerId":"player-1","kind":"OPENING",` +
			`"money":{"amount":"100.00","currency":"BRL"},` +
			`"balance":{"amount":"100.00","currency":"BRL"}}}`

		if string(result) != expected {
			t.Errorf("expected %s, got %s", expected, result)
		}
	})

	t.Run("rejects nil transaction", func(t *testing.T) {
		_, err := event.NewProcessed(metadata, nil)
		if !errors.Is(err, event.ErrInvalidData) {
			t.Fatalf("expected invalid data error, got: %v", err)
		}
	})

	t.Run("rejects pending transaction", func(t *testing.T) {
		amount, err := money.Parse("25.00", "BRL")
		if err != nil {
			t.Fatal(err)
		}

		transaction, err := wager.NewExternal(wager.ExternalInput{
			ID:                    "transaction-1",
			ProviderID:            "provider-a",
			ExternalTransactionID: "external-1",
			IdempotencyKey:        "key-1",
			PayloadHash:           strings.Repeat("a", 64),
			WalletID:              "wallet-1",
			PlayerID:              "player-1",
			RoundID:               "round-1",
			GameID:                "game-1",
			Kind:                  wager.Bet,
			Money:                 amount,
		}, now)
		if err != nil {
			t.Fatal(err)
		}

		_, err = event.NewProcessed(metadata, transaction)
		if !errors.Is(err, event.ErrInvalidData) {
			t.Fatalf("expected invalid data error, got: %v", err)
		}
	})
}
