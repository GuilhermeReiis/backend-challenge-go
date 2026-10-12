package repository

import (
	"context"
	"errors"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/service"
	"gorm.io/gorm"
)

type UnitOfWork struct {
	db *gorm.DB
}

var _ service.UnitOfWork = (*UnitOfWork)(nil)

func NewUnitOfWork(db *gorm.DB) *UnitOfWork {
	return &UnitOfWork{db: db}
}

func (u *UnitOfWork) Execute(ctx context.Context, operation func(service.Repositories) error) error {
	if operation == nil {
		return errors.New("postgres: transaction operation is required")
	}
	err := u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		session := newSession(tx, true)
		defer session.active.Store(false)

		repositories := service.Repositories{
			Wallets: &WalletRepository{session: session},
			Wagers:  &WagerRepository{session: session},
			Ledger:  &LedgerRepository{session: session},
			Inbox:   &InboxRepository{session: session},
			Outbox:  &OutboxRepository{session: session},
		}
		err := operation(repositories)
		if err != nil {
			return err
		}
		return ctx.Err()
	})
	return repositoryError(err)
}
