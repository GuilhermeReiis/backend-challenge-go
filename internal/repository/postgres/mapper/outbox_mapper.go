package mapper

import (
	"encoding/json"
	models "github.com/GuilhermeReiis/backend-challenge-go/internal/repository/postgres/model"
	"time"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/event"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/service"
)

func OutboxToModel(value service.OutboxEvent) (models.Outbox, error) {
	var envelope struct {
		EventID       string          `json:"eventId"`
		AggregateID   string          `json:"aggregateId"`
		EventType     string          `json:"eventType"`
		Version       int             `json:"version"`
		CorrelationID string          `json:"correlationId"`
		OccurredAt    time.Time       `json:"occurredAt"`
		Data          json.RawMessage `json:"data"`
	}
	err := json.Unmarshal(value.Payload, &envelope)
	if err != nil {
		return models.Outbox{}, event.ErrInvalidData
	}
	if envelope.EventID != value.EventID || envelope.AggregateID != value.AggregateID ||
		envelope.EventType != value.EventType || envelope.Version != value.Version ||
		envelope.CorrelationID != value.CorrelationID || !envelope.OccurredAt.Equal(value.OccurredAt) || value.OccurredAt.IsZero() {
		return models.Outbox{}, event.ErrInvalidMetadata
	}
	var data map[string]json.RawMessage
	err = json.Unmarshal(envelope.Data, &data)
	if err != nil || data == nil {
		return models.Outbox{}, event.ErrInvalidData
	}
	return models.Outbox{
		EventID: value.EventID, AggregateID: value.AggregateID, EventType: value.EventType,
		EventVersion: value.Version, CorrelationID: value.CorrelationID,
		Payload: append(json.RawMessage(nil), value.Payload...), OccurredAt: value.OccurredAt.UTC().Round(time.Microsecond),
	}, nil
}
