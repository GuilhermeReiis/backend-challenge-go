package repository

import (
	"context"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/repository/postgres/mapper"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/service"
	"gorm.io/gorm"
)

type OutboxRepository struct{ session *repositorySession }

var _ service.OutboxRepository = (*OutboxRepository)(nil)

func NewOutboxRepository(db *gorm.DB) *OutboxRepository {
	return &OutboxRepository{session: newSession(db, false)}
}

func (r *OutboxRepository) Append(ctx context.Context, value service.OutboxEvent) error {
	db, err := r.session.connection(ctx, true)
	if err != nil {
		return err
	}
	model, err := mapper.OutboxToModel(value)
	if err != nil {
		return err
	}
	return repositoryError(db.Create(&model).Error)
}
