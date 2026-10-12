package repository

import (
	"context"
	"errors"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/repository/postgres/mapper"
	models "github.com/GuilhermeReiis/backend-challenge-go/internal/repository/postgres/model"
	"time"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/service"
	"gorm.io/gorm"
)

type InboxRepository struct{ session *repositorySession }

var _ service.InboxRepository = (*InboxRepository)(nil)

func NewInboxRepository(db *gorm.DB) *InboxRepository {
	return &InboxRepository{session: newSession(db, false)}
}

func (r *InboxRepository) Find(ctx context.Context, consumerName string, messageID string) (service.InboxMessage, error) {
	db, err := r.session.connection(ctx, false)

	if err != nil {
		return service.InboxMessage{}, err
	}

	var model models.Inbox

	if err := db.Where("consumer_name = ? AND message_id = ?", consumerName, messageID).Take(&model).Error; err != nil {
		return service.InboxMessage{}, repositoryError(err)
	}

	return mapper.InboxFromModel(model), nil
}

func (r *InboxRepository) Create(ctx context.Context, message service.InboxMessage) error {
	db, err := r.session.connection(ctx, true)

	if err != nil {
		return err
	}

	model, err := mapper.InboxToModel(message)

	if err != nil {
		return err
	}

	return repositoryError(db.Create(&model).Error)
}

func (r *InboxRepository) Complete(ctx context.Context, consumerName string, messageID string, completedAt time.Time) error {
	db, err := r.session.connection(ctx, true)

	if err != nil {
		return err
	}

	if completedAt.IsZero() {
		return errors.New("postgres: inbox completion time is required")
	}

	result := db.Model(&models.Inbox{}).
		Where("consumer_name = ? AND message_id = ? AND completed_at IS NULL AND received_at <= ?", consumerName, messageID, completedAt).
		Update("completed_at", completedAt.UTC())

	if result.Error != nil {
		return repositoryError(result.Error)
	}

	if result.RowsAffected != 1 {
		return service.ErrConcurrentUpdate
	}

	return nil
}
