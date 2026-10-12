package repository

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/service"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type repositorySession struct {
	db          *gorm.DB
	transaction bool
	active      atomic.Bool
}

func newSession(db *gorm.DB, transaction bool) *repositorySession {
	session := &repositorySession{db: db, transaction: transaction}
	session.active.Store(true)
	return session
}

func (s *repositorySession) connection(ctx context.Context, write bool) (*gorm.DB, error) {
	if !s.active.Load() || (write && !s.transaction) {
		return nil, service.ErrTransactionRequired
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return s.db.WithContext(ctx), nil
}

func repositoryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.Join(service.ErrNotFound, err)
	}

	var pgError *pgconn.PgError

	if errors.As(err, &pgError) {
		switch pgError.Code {
		case "23505":
			return errors.Join(service.ErrAlreadyExists, err)
		case "40001", "40P01":
			return errors.Join(service.ErrConcurrentUpdate, err)
		}
	}

	return err
}
