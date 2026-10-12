//go:build integration

package repository_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/event"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/ledger"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/wager"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/wallet"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/repository/postgres/repository"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/service"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type openingRecords struct {
	wallet *wallet.Wallet
	wager  *wager.Transaction
	entry  ledger.Entry
	outbox service.OutboxEvent
	inbox  service.InboxMessage
}

func testMoney(t *testing.T, amount int64) money.Money {
	t.Helper()
	value, err := money.New(amount, "BRL")

	if err != nil {
		t.Fatal(err)
	}

	return value
}

func openingFixture(t *testing.T) openingRecords {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	balance := testMoney(t, 10000)
	w, err := wallet.New(testID(t), testID(t), balance, now)

	if err != nil {
		t.Fatal(err)
	}

	transaction, err := wager.NewOpening(wager.OpeningInput{ID: testID(t), WalletID: w.ID(), PlayerID: w.PlayerID(), Money: balance}, now)

	if err != nil {
		t.Fatal(err)
	}

	if err := transaction.Process(balance, "", now); err != nil {
		t.Fatal(err)
	}

	entry, err := ledger.New(testID(t), w.ID(), transaction.ID(), ledger.Credit, balance, testMoney(t, 0), balance, now)

	if err != nil {
		t.Fatal(err)
	}

	metadata := event.Metadata{EventID: testID(t), CorrelationID: testID(t), OccurredAt: now}
	envelope, err := event.NewProcessed(metadata, transaction)

	if err != nil {
		t.Fatal(err)
	}

	payload, err := json.Marshal(envelope)

	if err != nil {
		t.Fatal(err)
	}

	return openingRecords{
		wallet: w, wager: transaction, entry: entry,
		outbox: service.OutboxEvent{
			EventID: metadata.EventID, AggregateID: w.ID(), EventType: "WagerTransactionProcessed",
			Version: 1, CorrelationID: metadata.CorrelationID, OccurredAt: now, Payload: payload,
		},
		inbox: service.InboxMessage{ConsumerName: "test-consumer", MessageID: testID(t), PayloadHash: strings.Repeat("a", 64), ReceivedAt: now},
	}
}

func persistOpening(ctx context.Context, repositories service.Repositories, records openingRecords) error {
	if err := repositories.Wallets.Create(ctx, records.wallet); err != nil {
		return err
	}

	if err := repositories.Wagers.Create(ctx, records.wager); err != nil {
		return err
	}

	if err := repositories.Ledger.Append(ctx, records.entry); err != nil {
		return err
	}

	if err := repositories.Inbox.Create(ctx, records.inbox); err != nil {
		return err
	}

	if err := repositories.Inbox.Complete(ctx, records.inbox.ConsumerName, records.inbox.MessageID, records.inbox.ReceivedAt); err != nil {
		return err
	}

	return repositories.Outbox.Append(ctx, records.outbox)
}

func externalFixture(t *testing.T, w *wallet.Wallet, provider string, kind wager.Kind) *wager.Transaction {
	t.Helper()
	input := wager.ExternalInput{
		ID: testID(t), ProviderID: provider, ExternalTransactionID: testID(t), IdempotencyKey: testID(t),
		PayloadHash: strings.Repeat("a", 64), WalletID: w.ID(), PlayerID: w.PlayerID(), RoundID: "round-1", GameID: "game-1",
		Kind: kind, Money: testMoney(t, 100),
	}

	if kind == wager.Refund {
		input.ReferenceExternalTransactionID = "missing-bet"
	}

	value, err := wager.NewExternal(input, w.CreatedAt())

	if err != nil {
		t.Fatal(err)
	}

	return value
}

func countRows(t *testing.T, db *gorm.DB, table string, column string, id string) int64 {
	t.Helper()
	var count int64

	if err := db.Table(table).Where(column+" = ?", id).Count(&count).Error; err != nil {
		t.Fatal(err)
	}

	return count
}

func TestRepositories(t *testing.T) {
	db, admin := repositoryTestDatabase(t)
	uow := repository.NewUnitOfWork(db)
	ctx := context.Background()

	t.Run("commits all repositories and restores exact domain state", func(t *testing.T) {
		records := openingFixture(t)

		if err := uow.Execute(ctx, func(repositories service.Repositories) error { return persistOpening(ctx, repositories, records) }); err != nil {
			t.Fatal(err)
		}

		w, err := repository.NewWalletRepository(db).FindByID(ctx, records.wallet.ID())

		if err != nil {
			t.Fatal(err)
		}

		if *w != *records.wallet {
			t.Error("expected wallet state to survive persistence exactly")
		}

		transaction, err := repository.NewWagerRepository(db).FindByID(ctx, records.wager.ID())

		if err != nil {
			t.Fatal(err)
		}

		if *transaction != *records.wager {
			t.Error("expected internal transaction and nullable fields to survive persistence")
		}

		page, err := repository.NewLedgerRepository(db).List(ctx, w.ID(), service.LedgerQuery{Limit: 1})

		if err != nil {
			t.Fatal(err)
		}

		if len(page) != 1 || page[0].Entry != records.entry || page[0].Sequence < 1 {
			t.Fatal("expected persisted ledger entry and database sequence")
		}

		next, err := repository.NewLedgerRepository(db).List(ctx, w.ID(), service.LedgerQuery{AfterSequence: page[0].Sequence, Limit: 1})

		if err != nil {
			t.Fatal(err)
		}

		if len(next) != 0 {
			t.Error("expected cursor pagination not to repeat the entry")
		}

		inbox, err := repository.NewInboxRepository(db).Find(ctx, records.inbox.ConsumerName, records.inbox.MessageID)

		if err != nil {
			t.Fatal(err)
		}

		if inbox.CompletedAt.IsZero() || inbox.PayloadHash != records.inbox.PayloadHash {
			t.Error("expected inbox completion in the same commit")
		}

		if countRows(t, db, "outbox_events", "event_id", records.outbox.EventID) != 1 {
			t.Error("expected committed outbox snapshot")
		}
	})

	t.Run("rolls back every repository and invalidates transaction repositories", func(t *testing.T) {
		records := openingFixture(t)
		failure := errors.New("test rollback")
		var escaped service.WalletRepository

		if err := uow.Execute(ctx, func(repositories service.Repositories) error {
			escaped = repositories.Wallets

			if err := persistOpening(ctx, repositories, records); err != nil {
				return err
			}

			return failure
		}); !errors.Is(err, failure) {
			t.Fatal(err)
		}

		for _, row := range []struct{ table, column, id string }{
			{"wallets", "id", records.wallet.ID()}, {"wager_transactions", "id", records.wager.ID()},
			{"wallet_ledger_entries", "id", records.entry.ID()}, {"inbox_messages", "message_id", records.inbox.MessageID},
			{"outbox_events", "event_id", records.outbox.EventID},
		} {
			if countRows(t, db, row.table, row.column, row.id) != 0 {
				t.Errorf("expected rollback of %s", row.table)
			}
		}

		if _, err := escaped.FindByID(ctx, records.wallet.ID()); !errors.Is(err, service.ErrTransactionRequired) {
			t.Error("expected expired repository to reject reuse")
		}
	})

	t.Run("requires transaction for writes and locks", func(t *testing.T) {
		records := openingFixture(t)
		repository := repository.NewWalletRepository(db)

		if err := repository.Create(ctx, records.wallet); !errors.Is(err, service.ErrTransactionRequired) {
			t.Error("expected write outside transaction to fail")
		}

		if _, err := repository.FindByIDForUpdate(ctx, records.wallet.ID()); !errors.Is(err, service.ErrTransactionRequired) {
			t.Error("expected lock outside transaction to fail")
		}

		if _, err := repository.FindByID(ctx, records.wallet.ID()); !errors.Is(err, service.ErrNotFound) {
			t.Error("expected missing record to be classified")
		}
	})

	t.Run("persists zero balance and rejects stale versions", func(t *testing.T) {
		records := openingFixture(t)

		if err := uow.Execute(ctx, func(repositories service.Repositories) error { return persistOpening(ctx, repositories, records) }); err != nil {
			t.Fatal(err)
		}

		if err := uow.Execute(ctx, func(repositories service.Repositories) error {
			w, err := repositories.Wallets.FindByIDForUpdate(ctx, records.wallet.ID())

			if err != nil {
				return err
			}

			if err := w.Debit(w.Balance(), w.UpdatedAt()); err != nil {
				return err
			}

			return repositories.Wallets.Update(ctx, w, 1)
		}); err != nil {
			t.Fatal(err)
		}

		w, err := repository.NewWalletRepository(db).FindByID(ctx, records.wallet.ID())

		if err != nil {
			t.Fatal(err)
		}

		if w.Balance().Amount() != 0 || w.Version() != 2 {
			t.Error("expected zero balance and incremented version")
		}

		if err := records.wallet.Debit(records.wallet.Balance(), records.wallet.UpdatedAt()); err != nil {
			t.Fatal(err)
		}

		if err := uow.Execute(ctx, func(repositories service.Repositories) error {
			return repositories.Wallets.Update(ctx, records.wallet, 1)
		}); !errors.Is(err, service.ErrConcurrentUpdate) {
			t.Error("expected stale update to fail")
		}
	})

	t.Run("preserves terminal transactions and ledger immutability", func(t *testing.T) {
		records := openingFixture(t)

		if err := uow.Execute(ctx, func(repositories service.Repositories) error { return persistOpening(ctx, repositories, records) }); err != nil {
			t.Fatal(err)
		}

		if err := uow.Execute(ctx, func(repositories service.Repositories) error { return repositories.Wagers.Update(ctx, records.wager) }); !errors.Is(err, wager.ErrTerminalState) {
			t.Error("expected persisted terminal transaction to reject updates")
		}

		for _, statement := range []string{
			"UPDATE wallet_ledger_entries SET balance_after_minor = balance_after_minor WHERE id = ?",
			"DELETE FROM wallet_ledger_entries WHERE id = ?",
		} {
			var pgError *pgconn.PgError

			if err := admin.Exec(statement, records.entry.ID()).Error; !errors.As(err, &pgError) || pgError.Code != "23001" {
				t.Error("expected append-only trigger to reject mutation")
			}
		}
	})

	t.Run("persists pending reference schedule and rejection with null result", func(t *testing.T) {
		records := openingFixture(t)
		transaction := externalFixture(t, records.wallet, "provider-a", wager.Refund)

		if err := uow.Execute(ctx, func(repositories service.Repositories) error {
			if err := repositories.Wallets.Create(ctx, records.wallet); err != nil {
				return err
			}

			if err := repositories.Wagers.Create(ctx, transaction); err != nil {
				return err
			}

			if err := transaction.AwaitReference(transaction.CreatedAt()); err != nil {
				return err
			}

			return repositories.Wagers.SavePendingReference(ctx, transaction, service.ReferenceSchedule{
				Attempts: 2, NextAttemptAt: transaction.UpdatedAt().Add(time.Minute), ExpiresAt: transaction.CreatedAt().Add(time.Hour),
			})
		}); err != nil {
			t.Fatal(err)
		}

		loaded, err := repository.NewWagerRepository(db).FindByExternalID(ctx, "provider-a", transaction.ExternalTransactionID())

		if err != nil {
			t.Fatal(err)
		}

		if loaded.Status() != wager.PendingReference {
			t.Fatal("expected pending reference state")
		}

		if err := loaded.Reject("REFERENCE_NOT_FOUND", loaded.UpdatedAt().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}

		if err := uow.Execute(ctx, func(repositories service.Repositories) error { return repositories.Wagers.Update(ctx, loaded) }); err != nil {
			t.Fatal(err)
		}

		loaded, err = repository.NewWagerRepository(db).FindByIdempotencyKey(ctx, "provider-a", transaction.IdempotencyKey())

		if err != nil {
			t.Fatal(err)
		}

		_, hasResult := loaded.ResultBalance()

		if loaded.Status() != wager.Rejected || hasResult || loaded.FailureCode() != "REFERENCE_NOT_FOUND" {
			t.Error("expected stable rejection and no financial result")
		}

		if _, err := repository.NewWagerRepository(db).FindByExternalID(ctx, "provider-b", transaction.ExternalTransactionID()); !errors.Is(err, service.ErrNotFound) {
			t.Error("expected lookup to be scoped to provider")
		}
	})

	t.Run("classifies duplicate operations and rolls back related writes", func(t *testing.T) {
		records := openingFixture(t)
		transaction := externalFixture(t, records.wallet, "provider-a", wager.Bet)

		if err := uow.Execute(ctx, func(repositories service.Repositories) error {
			if err := repositories.Wallets.Create(ctx, records.wallet); err != nil {
				return err
			}

			return repositories.Wagers.Create(ctx, transaction)
		}); err != nil {
			t.Fatal(err)
		}

		message := records.inbox

		if err := uow.Execute(ctx, func(repositories service.Repositories) error {
			if err := repositories.Inbox.Create(ctx, message); err != nil {
				return err
			}

			return repositories.Wagers.Create(ctx, transaction)
		}); !errors.Is(err, service.ErrAlreadyExists) {
			t.Fatal("expected duplicate operation error")
		}

		if countRows(t, db, "inbox_messages", "message_id", message.MessageID) != 0 {
			t.Error("expected failed transaction to roll back inbox")
		}
	})

	t.Run("rejects a transaction for a different wallet player", func(t *testing.T) {
		records := openingFixture(t)
		input := wager.ExternalInput{
			ID: testID(t), ProviderID: "provider-a", ExternalTransactionID: testID(t), IdempotencyKey: testID(t),
			PayloadHash: strings.Repeat("a", 64), WalletID: records.wallet.ID(), PlayerID: testID(t),
			RoundID: "round-1", GameID: "game-1", Kind: wager.Bet, Money: testMoney(t, 100),
		}
		transaction, err := wager.NewExternal(input, records.wallet.CreatedAt())

		if err != nil {
			t.Fatal(err)
		}

		if err := uow.Execute(ctx, func(repositories service.Repositories) error {
			if err := repositories.Wallets.Create(ctx, records.wallet); err != nil {
				return err
			}

			return repositories.Wagers.Create(ctx, transaction)
		}); !errors.Is(err, wager.ErrInvalidIdentity) {
			t.Fatal("expected wallet ownership mismatch to fail")
		}

		if countRows(t, db, "wallets", "id", records.wallet.ID()) != 0 {
			t.Error("expected identity mismatch to roll back the transaction")
		}
	})

	t.Run("preserves outbox payload with submicrosecond occurrence time", func(t *testing.T) {
		records := openingFixture(t)
		records.outbox.OccurredAt = records.outbox.OccurredAt.Add(789 * time.Nanosecond)
		envelope, err := event.NewProcessed(event.Metadata{
			EventID: records.outbox.EventID, CorrelationID: records.outbox.CorrelationID, OccurredAt: records.outbox.OccurredAt,
		}, records.wager)

		if err != nil {
			t.Fatal(err)
		}

		records.outbox.Payload, err = json.Marshal(envelope)

		if err != nil {
			t.Fatal(err)
		}

		if err := uow.Execute(ctx, func(repositories service.Repositories) error {
			return persistOpening(ctx, repositories, records)
		}); err != nil {
			t.Fatal(err)
		}

		var stored struct {
			OccurredAt time.Time
			Payload    json.RawMessage
		}

		if err := db.Table("outbox_events").Select("occurred_at, payload").Where("event_id = ?", records.outbox.EventID).Take(&stored).Error; err != nil {
			t.Fatal(err)
		}

		var payload struct {
			OccurredAt time.Time `json:"occurredAt"`
		}

		if err := json.Unmarshal(stored.Payload, &payload); err != nil {
			t.Fatal(err)
		}

		if !payload.OccurredAt.Equal(records.outbox.OccurredAt) || !stored.OccurredAt.Equal(records.outbox.OccurredAt.Round(time.Microsecond)) {
			t.Error("expected exact payload occurrence and PostgreSQL timestamp precision")
		}
	})

	t.Run("rejects an outbox envelope with inconsistent metadata", func(t *testing.T) {
		records := openingFixture(t)
		records.outbox.CorrelationID = "different-correlation"

		if err := uow.Execute(ctx, func(repositories service.Repositories) error { return persistOpening(ctx, repositories, records) }); !errors.Is(err, event.ErrInvalidMetadata) {
			t.Fatal("expected inconsistent snapshot to fail")
		}

		if countRows(t, db, "wallets", "id", records.wallet.ID()) != 0 {
			t.Error("expected invalid outbox to roll back wallet creation")
		}
	})

	t.Run("locks one wallet without blocking another wallet", func(t *testing.T) {
		first, second := openingFixture(t), openingFixture(t)

		if err := uow.Execute(ctx, func(repositories service.Repositories) error {
			if err := repositories.Wallets.Create(ctx, first.wallet); err != nil {
				return err
			}

			return repositories.Wallets.Create(ctx, second.wallet)
		}); err != nil {
			t.Fatal(err)
		}

		locked, release := make(chan struct{}), make(chan struct{})
		var once sync.Once
		unlock := func() { once.Do(func() { close(release) }) }
		t.Cleanup(unlock)
		finished := make(chan error, 1)
		lockCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		go func() {
			finished <- uow.Execute(lockCtx, func(repositories service.Repositories) error {
				if _, err := repositories.Wallets.FindByIDForUpdate(lockCtx, first.wallet.ID()); err != nil {
					return err
				}

				close(locked)
				select {
				case <-release:
					return nil
				case <-lockCtx.Done():
					return lockCtx.Err()
				}
			})
		}()
		select {
		case <-locked:
		case <-lockCtx.Done():
			t.Fatal("first transaction did not acquire the wallet lock")
		}
		blockedCtx, blockedCancel := context.WithTimeout(ctx, 150*time.Millisecond)
		defer blockedCancel()

		if err := uow.Execute(blockedCtx, func(repositories service.Repositories) error {
			_, err := repositories.Wallets.FindByIDForUpdate(blockedCtx, first.wallet.ID())
			return err
		}); err == nil {
			t.Error("expected another writer on the same wallet to wait")
		}

		if err := uow.Execute(lockCtx, func(repositories service.Repositories) error {
			_, err := repositories.Wallets.FindByIDForUpdate(lockCtx, second.wallet.ID())
			return err
		}); err != nil {
			t.Fatal("expected an independent wallet to remain available")
		}

		unlock()

		if err := <-finished; err != nil {
			t.Fatal(err)
		}
	})
}
