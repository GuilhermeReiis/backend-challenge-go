package service

import "context"

// Repositories are bound to one SQL transaction and are valid only during Execute.
type Repositories struct {
	Wallets WalletRepository
	Wagers  WagerRepository
	Ledger  LedgerRepository
	Inbox   InboxRepository
	Outbox  OutboxRepository
}

type UnitOfWork interface {
	// Execute commits when operation returns nil and rolls back when it returns an error.
	// It reports commit failures and respects context cancellation.
	// Repositories must not escape the callback or be used concurrently.
	Execute(ctx context.Context, operation func(Repositories) error) error
}
