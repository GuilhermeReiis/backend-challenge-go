package service

import (
	"encoding/json"
	"time"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/ledger"
)

// ReferenceSchedule stores retry metadata separately from the financial entity.
type ReferenceSchedule struct {
	Attempts      int
	NextAttemptAt time.Time
	ExpiresAt     time.Time
}

// LedgerQuery uses a sequence cursor internally; HTTP encodes it as an opaque token.
type LedgerQuery struct {
	AfterSequence int64
	Limit         int
}

type LedgerRecord struct {
	Sequence int64
	Entry    ledger.Entry
}

type InboxMessage struct {
	ConsumerName string
	MessageID    string
	PayloadHash  string
	ReceivedAt   time.Time
	// A zero CompletedAt represents an unfinished message.
	CompletedAt time.Time
}

// OutboxEvent carries the envelope metadata and the already serialized typed payload.
// The repository must verify that the metadata agrees with the payload.
type OutboxEvent struct {
	EventID       string
	AggregateID   string
	EventType     string
	Version       int
	CorrelationID string
	OccurredAt    time.Time
	Payload       json.RawMessage
}
