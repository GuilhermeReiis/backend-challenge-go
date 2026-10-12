package repository

import (
	"context"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/repository/postgres/mapper"
	models "github.com/GuilhermeReiis/backend-challenge-go/internal/repository/postgres/model"
	"math"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/wallet"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/service"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WalletRepository struct{ session *repositorySession }

var _ service.WalletRepository = (*WalletRepository)(nil)

func NewWalletRepository(db *gorm.DB) *WalletRepository {
	return &WalletRepository{session: newSession(db, false)}
}

func (r *WalletRepository) Create(ctx context.Context, value *wallet.Wallet) error {
	db, err := r.session.connection(ctx, true)

	if err != nil {
		return err
	}

	model, err := mapper.WalletToModel(value)

	if err != nil {
		return err
	}

	if model.Version != 1 {
		return wallet.ErrInvalidVersion
	}

	return repositoryError(db.Create(&model).Error)
}

func (r *WalletRepository) FindByID(ctx context.Context, id string) (*wallet.Wallet, error) {
	return r.find(ctx, id, false)
}

func (r *WalletRepository) FindByIDForUpdate(ctx context.Context, id string) (*wallet.Wallet, error) {
	return r.find(ctx, id, true)
}

func (r *WalletRepository) find(ctx context.Context, id string, lock bool) (*wallet.Wallet, error) {
	db, err := r.session.connection(ctx, lock)

	if err != nil {
		return nil, err
	}

	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}

	var model models.Wallet

	if err := db.Where("id = ?", id).Take(&model).Error; err != nil {
		return nil, repositoryError(err)
	}

	return mapper.WalletFromModel(model)
}

func (r *WalletRepository) Update(ctx context.Context, value *wallet.Wallet, expectedVersion int64) error {
	db, err := r.session.connection(ctx, true)

	if err != nil {
		return err
	}

	model, err := mapper.WalletToModel(value)

	if err != nil {
		return err
	}

	if expectedVersion < 1 || expectedVersion == math.MaxInt64 || model.Version != expectedVersion+1 {
		return service.ErrConcurrentUpdate
	}

	result := db.Model(&models.Wallet{}).
		Where("id = ? AND player_id = ? AND currency = ? AND version = ? AND created_at = ? AND updated_at <= ? AND balance_minor <> ?",
			model.ID, model.PlayerID, model.Currency, expectedVersion, model.CreatedAt, model.UpdatedAt, model.BalanceMinor).
		Updates(map[string]any{
			"balance_minor": model.BalanceMinor, "version": model.Version, "updated_at": model.UpdatedAt,
		})

	if result.Error != nil {
		return repositoryError(result.Error)
	}

	if result.RowsAffected != 1 {
		return service.ErrConcurrentUpdate
	}

	return nil
}
