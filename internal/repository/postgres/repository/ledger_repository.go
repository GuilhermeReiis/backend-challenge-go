package repository

import (
	"context"
	"errors"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/repository/postgres/mapper"
	models "github.com/GuilhermeReiis/backend-challenge-go/internal/repository/postgres/model"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/ledger"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/wager"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/service"
	"gorm.io/gorm"
)

type LedgerRepository struct{ session *repositorySession }

var _ service.LedgerRepository = (*LedgerRepository)(nil)

func NewLedgerRepository(db *gorm.DB) *LedgerRepository {
	return &LedgerRepository{session: newSession(db, false)}
}

func (r *LedgerRepository) Append(ctx context.Context, entry ledger.Entry) error {
	db, err := r.session.connection(ctx, true)

	if err != nil {
		return err
	}

	model, err := mapper.LedgerToModel(entry)

	if err != nil {
		return err
	}

	transaction, err := (&WagerRepository{session: r.session}).FindByID(ctx, entry.TransactionID())

	if err != nil {
		return err
	}

	if transaction.Status() != wager.Processed || transaction.WalletID() != entry.WalletID() || transaction.Money() != entry.Money() {
		return ledger.ErrInconsistentBalance
	}

	result, _ := transaction.ResultBalance()

	if result != entry.BalanceAfter() {
		return ledger.ErrInconsistentBalance
	}

	direction := ledger.Credit
	switch transaction.Kind() {
	case wager.Bet:
		direction = ledger.Debit
	case wager.Opening, wager.Win, wager.Refund:
	case wager.Rollback:
		reference, err := (&WagerRepository{session: r.session}).FindByID(ctx, transaction.ReferenceTransactionID())

		if err != nil {
			return err
		}

		switch reference.Kind() {
		case wager.Bet:
		case wager.Win, wager.Refund:
			direction = ledger.Debit
		default:
			return ledger.ErrInvalidDirection
		}
	default:
		return ledger.ErrInvalidDirection
	}

	if entry.Direction() != direction {
		return ledger.ErrInvalidDirection
	}

	return repositoryError(db.Create(&model).Error)
}

func (r *LedgerRepository) List(ctx context.Context, walletID string, query service.LedgerQuery) ([]service.LedgerRecord, error) {
	db, err := r.session.connection(ctx, false)

	if err != nil {
		return nil, err
	}

	if query.AfterSequence < 0 || query.Limit < 1 || query.Limit > 100 {
		return nil, errors.New("postgres: invalid ledger pagination")
	}

	models := []models.Ledger{}

	if err := db.Where("wallet_id = ? AND sequence > ?", walletID, query.AfterSequence).Order("sequence ASC").Limit(query.Limit).Find(&models).Error; err != nil {
		return nil, repositoryError(err)
	}

	records := make([]service.LedgerRecord, 0, len(models))
	for _, model := range models {
		entry, err := mapper.LedgerFromModel(model)

		if err != nil {
			return nil, err
		}

		records = append(records, service.LedgerRecord{Sequence: model.Sequence, Entry: entry})
	}
	return records, nil
}
