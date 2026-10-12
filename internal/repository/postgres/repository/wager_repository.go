package repository

import (
	"context"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/repository/postgres/mapper"
	models "github.com/GuilhermeReiis/backend-challenge-go/internal/repository/postgres/model"
	"math"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/wager"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/service"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WagerRepository struct{ session *repositorySession }

var _ service.WagerRepository = (*WagerRepository)(nil)

func NewWagerRepository(db *gorm.DB) *WagerRepository {
	return &WagerRepository{session: newSession(db, false)}
}

func (r *WagerRepository) Create(ctx context.Context, value *wager.Transaction) error {
	db, err := r.session.connection(ctx, true)

	if err != nil {
		return err
	}

	model, err := mapper.WagerToModel(value)

	if err != nil {
		return err
	}

	if value.Status() == wager.PendingReference {
		return wager.ErrInvalidTransaction
	}

	var matches int64

	if err := db.Model(&models.Wallet{}).Where("id = ? AND player_id = ? AND currency = ?", model.WalletID, model.PlayerID, model.Currency).Count(&matches).Error; err != nil {
		return repositoryError(err)
	}

	if matches != 1 {
		return wager.ErrInvalidIdentity
	}

	return repositoryError(db.Create(&model).Error)
}

func (r *WagerRepository) FindByID(ctx context.Context, id string) (*wager.Transaction, error) {
	return r.find(ctx, "id = ?", id)
}

func (r *WagerRepository) FindByExternalID(ctx context.Context, providerID string, externalID string) (*wager.Transaction, error) {
	return r.find(ctx, "origin = 'EXTERNAL' AND provider_id = ? AND external_transaction_id = ?", providerID, externalID)
}

func (r *WagerRepository) FindByIdempotencyKey(ctx context.Context, providerID string, key string) (*wager.Transaction, error) {
	return r.find(ctx, "origin = 'EXTERNAL' AND provider_id = ? AND idempotency_key = ?", providerID, key)
}

func (r *WagerRepository) find(ctx context.Context, query string, args ...any) (*wager.Transaction, error) {
	db, err := r.session.connection(ctx, false)

	if err != nil {
		return nil, err
	}

	var model models.Wager

	if err := db.Where(query, args...).Take(&model).Error; err != nil {
		return nil, repositoryError(err)
	}

	return mapper.WagerFromModel(model)
}

func (r *WagerRepository) Update(ctx context.Context, value *wager.Transaction) error {
	if value != nil && value.Status() == wager.PendingReference {
		return wager.ErrInvalidTransaction
	}

	return r.update(ctx, value, nil)
}

func (r *WagerRepository) SavePendingReference(ctx context.Context, value *wager.Transaction, schedule service.ReferenceSchedule) error {
	if value == nil || value.Status() != wager.PendingReference {
		return wager.ErrInvalidTransaction
	}

	if schedule.Attempts < 0 || schedule.Attempts > math.MaxInt32 || schedule.NextAttemptAt.IsZero() ||
		schedule.NextAttemptAt.Before(value.UpdatedAt()) || !schedule.ExpiresAt.After(value.CreatedAt()) {
		return wager.ErrInvalidTime
	}

	return r.update(ctx, value, &schedule)
}

func (r *WagerRepository) update(ctx context.Context, value *wager.Transaction, schedule *service.ReferenceSchedule) error {
	db, err := r.session.connection(ctx, true)

	if err != nil {
		return err
	}

	model, err := mapper.WagerToModel(value)

	if err != nil {
		return err
	}

	var current models.Wager

	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", model.ID).Take(&current).Error; err != nil {
		return repositoryError(err)
	}

	if current.Status == string(wager.Processed) || current.Status == string(wager.Rejected) || current.Status == string(wager.Failed) {
		return wager.ErrTerminalState
	}

	if current.Status == string(wager.PendingReference) && model.Status == string(wager.Pending) {
		return wager.ErrInvalidTransaction
	}

	if !sameWagerIdentity(current, model) || model.UpdatedAt.Before(current.UpdatedAt) {
		return wager.ErrInvalidTransaction
	}

	updates := map[string]any{
		"status": model.Status, "reference_transaction_id": model.ReferenceTransactionID,
		"failure_code": model.FailureCode, "result_balance_minor": model.ResultBalanceMinor,
		"processed_at": model.ProcessedAt, "updated_at": model.UpdatedAt, "next_attempt_at": nil,
	}

	if schedule != nil {
		updates["attempts"] = schedule.Attempts
		updates["next_attempt_at"] = mapper.OptionalTime(schedule.NextAttemptAt)
		updates["expires_at"] = mapper.OptionalTime(schedule.ExpiresAt)
	}

	result := db.Model(&models.Wager{}).Where("id = ? AND status IN ?", model.ID, []string{string(wager.Pending), string(wager.PendingReference)}).Updates(updates)

	if result.Error != nil {
		return repositoryError(result.Error)
	}

	if result.RowsAffected != 1 {
		return service.ErrConcurrentUpdate
	}

	return nil
}

func sameWagerIdentity(left models.Wager, right models.Wager) bool {
	return left.ID == right.ID && left.Origin == right.Origin && left.Kind == right.Kind &&
		left.WalletID == right.WalletID && left.PlayerID == right.PlayerID && left.Currency == right.Currency &&
		left.AmountMinor == right.AmountMinor && left.CreatedAt.Equal(right.CreatedAt) &&
		mapper.StringValue(left.ProviderID) == mapper.StringValue(right.ProviderID) &&
		mapper.StringValue(left.ExternalTransactionID) == mapper.StringValue(right.ExternalTransactionID) &&
		mapper.StringValue(left.IdempotencyKey) == mapper.StringValue(right.IdempotencyKey) &&
		mapper.StringValue(left.PayloadHash) == mapper.StringValue(right.PayloadHash) &&
		mapper.StringValue(left.RoundID) == mapper.StringValue(right.RoundID) && mapper.StringValue(left.GameID) == mapper.StringValue(right.GameID) &&
		mapper.StringValue(left.ReferenceExternalTransactionID) == mapper.StringValue(right.ReferenceExternalTransactionID)
}
