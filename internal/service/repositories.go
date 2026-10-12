package service

import (
	"context"
	"time"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/ledger"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/wager"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/wallet"
)

// Repository writes must use the repositories supplied by a UnitOfWork.
type WalletRepository interface {
	Create(ctx context.Context, value *wallet.Wallet) error
	FindByID(ctx context.Context, id string) (*wallet.Wallet, error)
	// FindByIDForUpdate holds a wallet lock until the transaction ends.
	FindByIDForUpdate(ctx context.Context, id string) (*wallet.Wallet, error)
	// Update must check expectedVersion and report ErrConcurrentUpdate on a mismatch.
	Update(ctx context.Context, value *wallet.Wallet, expectedVersion int64) error
}

type WagerRepository interface {
	Create(ctx context.Context, value *wager.Transaction) error
	FindByID(ctx context.Context, id string) (*wager.Transaction, error)
	FindByExternalID(ctx context.Context, providerID string, externalID string) (*wager.Transaction, error)
	FindByIdempotencyKey(ctx context.Context, providerID string, key string) (*wager.Transaction, error)
	// Update must preserve terminal transactions and their original result.
	Update(ctx context.Context, value *wager.Transaction) error
	// SavePendingReference persists the pending state and its durable retry schedule together.
	SavePendingReference(ctx context.Context, value *wager.Transaction, schedule ReferenceSchedule) error
}

type LedgerRepository interface {
	Append(ctx context.Context, entry ledger.Entry) error
	// List returns entries in ascending sequence order after the supplied cursor.
	List(ctx context.Context, walletID string, query LedgerQuery) ([]LedgerRecord, error)
}

type InboxRepository interface {
	Find(ctx context.Context, consumerName string, messageID string) (InboxMessage, error)
	Create(ctx context.Context, message InboxMessage) error
	Complete(ctx context.Context, consumerName string, messageID string, completedAt time.Time) error
}

type OutboxRepository interface {
	// Append persists a snapshot; publishing belongs to the outbox worker after commit.
	Append(ctx context.Context, event OutboxEvent) error
}
