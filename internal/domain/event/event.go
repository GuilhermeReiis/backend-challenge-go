package event

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidMetadata = errors.New("event: invalid metadata")
	ErrInvalidData     = errors.New("event: invalid data")
)

type Metadata struct {
	EventID       string
	CorrelationID string
	CausationID   string
	OccurredAt    time.Time
}

type Envelope[T any] struct {
	eventID       string
	eventType     string
	aggregateID   string
	correlationID string
	causationID   string
	occurredAt    time.Time
	version       int
	data          T
}

func newEnvelope[T any](
	metadata Metadata,
	eventType string,
	aggregateID string,
	data T,
) (Envelope[T], error) {
	if strings.TrimSpace(metadata.EventID) == "" ||
		strings.TrimSpace(metadata.CorrelationID) == "" ||
		strings.TrimSpace(aggregateID) == "" ||
		metadata.OccurredAt.IsZero() {
		return Envelope[T]{}, ErrInvalidMetadata
	}

	if metadata.CausationID != "" &&
		strings.TrimSpace(metadata.CausationID) == "" {
		return Envelope[T]{}, ErrInvalidMetadata
	}

	return Envelope[T]{
		eventID:       metadata.EventID,
		eventType:     eventType,
		aggregateID:   aggregateID,
		correlationID: metadata.CorrelationID,
		causationID:   metadata.CausationID,
		occurredAt:    metadata.OccurredAt.UTC(),
		version:       1,
		data:          data,
	}, nil
}

func (e Envelope[T]) MarshalJSON() ([]byte, error) {
	if e.eventID == "" || e.version != 1 {
		return nil, ErrInvalidMetadata
	}

	return json.Marshal(struct {
		EventID       string    `json:"eventId"`
		EventType     string    `json:"eventType"`
		AggregateID   string    `json:"aggregateId"`
		CorrelationID string    `json:"correlationId"`
		CausationID   string    `json:"causationId,omitempty"`
		OccurredAt    time.Time `json:"occurredAt"`
		Version       int       `json:"version"`
		Data          T         `json:"data"`
	}{
		EventID:       e.eventID,
		EventType:     e.eventType,
		AggregateID:   e.aggregateID,
		CorrelationID: e.correlationID,
		CausationID:   e.causationID,
		OccurredAt:    e.occurredAt,
		Version:       e.version,
		Data:          e.data,
	})
}
