package model

import (
	"encoding/json"
	"time"
)

type Outbox struct {
	EventID       string          `gorm:"column:event_id;primaryKey"`
	AggregateID   string          `gorm:"column:aggregate_id"`
	EventType     string          `gorm:"column:event_type"`
	EventVersion  int             `gorm:"column:event_version"`
	CorrelationID string          `gorm:"column:correlation_id"`
	Payload       json.RawMessage `gorm:"column:payload;type:jsonb"`
	OccurredAt    time.Time       `gorm:"column:occurred_at"`
}

func (Outbox) TableName() string { return "outbox_events" }
