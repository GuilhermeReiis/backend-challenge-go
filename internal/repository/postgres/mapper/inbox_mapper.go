package mapper

import (
	"errors"
	models "github.com/GuilhermeReiis/backend-challenge-go/internal/repository/postgres/model"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/service"
)

func InboxFromModel(model models.Inbox) service.InboxMessage {
	return service.InboxMessage{
		ConsumerName: model.ConsumerName,
		MessageID:    model.MessageID,
		PayloadHash:  model.PayloadHash,
		ReceivedAt:   model.ReceivedAt,
		CompletedAt:  timeValue(model.CompletedAt),
	}
}

func InboxToModel(message service.InboxMessage) (models.Inbox, error) {
	if message.ReceivedAt.IsZero() {
		return models.Inbox{}, errors.New("postgres: inbox received time is required")
	}

	return models.Inbox{
		ConsumerName: message.ConsumerName,
		MessageID:    message.MessageID,
		PayloadHash:  message.PayloadHash,
		ReceivedAt:   databaseTime(message.ReceivedAt),
		CompletedAt:  OptionalTime(message.CompletedAt),
	}, nil
}
