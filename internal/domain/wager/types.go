package wager

type Kind string

const (
	Opening  Kind = "OPENING"
	Bet      Kind = "BET"
	Win      Kind = "WIN"
	Loss     Kind = "LOSS"
	Refund   Kind = "REFUND"
	Rollback Kind = "ROLLBACK"
)

type Status string

const (
	Pending          Status = "PENDING"
	PendingReference Status = "PENDING_REFERENCE"
	Processed        Status = "PROCESSED"
	Rejected         Status = "REJECTED"
	Failed           Status = "FAILED"
)

type Origin string

const (
	InternalOrigin Origin = "INTERNAL"
	ExternalOrigin Origin = "EXTERNAL"
)
