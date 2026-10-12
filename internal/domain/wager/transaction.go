package wager

import (
	"errors"
	"strings"
	"time"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
)

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

var (
	ErrInvalidIdentity    = errors.New("wager: invalid identity")
	ErrInvalidKind        = errors.New("wager: invalid external kind")
	ErrInvalidAmount      = errors.New("wager: invalid amount for kind")
	ErrInvalidReference   = errors.New("wager: invalid reference")
	ErrInvalidTime        = errors.New("wager: invalid time")
	ErrInvalidTransaction = errors.New("wager: uninitialized transaction")
	ErrTerminalState      = errors.New("wager: terminal state")
	ErrInvalidFailureCode = errors.New("wager: invalid failure code")
)

type ExternalInput struct {
	ID                             string
	ProviderID                     string
	ExternalTransactionID          string
	IdempotencyKey                 string
	PayloadHash                    string
	WalletID                       string
	PlayerID                       string
	RoundID                        string
	GameID                         string
	Kind                           Kind
	Money                          money.Money
	ReferenceExternalTransactionID string
}

type Transaction struct {
	id                             string
	providerID                     string
	externalTransactionID          string
	idempotencyKey                 string
	payloadHash                    string
	walletID                       string
	playerID                       string
	roundID                        string
	gameID                         string
	kind                           Kind
	amount                         money.Money
	referenceExternalTransactionID string
	status                         Status
	failureCode                    string
	resultBalance                  money.Money
	processedAt                    time.Time
	referenceTransactionID         string
	origin                         Origin
	createdAt                      time.Time
	updatedAt                      time.Time
}

type OpeningInput struct {
	ID       string
	WalletID string
	PlayerID string
	Money    money.Money
}

func NewExternal(input ExternalInput, now time.Time) (*Transaction, error) {
	required := []string{
		input.ID,
		input.ProviderID,
		input.ExternalTransactionID,
		input.IdempotencyKey,
		input.PayloadHash,
		input.WalletID,
		input.PlayerID,
		input.RoundID,
		input.GameID,
	}

	for _, value := range required {
		if strings.TrimSpace(value) == "" {
			return nil, ErrInvalidIdentity
		}
	}

	if now.IsZero() {
		return nil, ErrInvalidTime
	}

	switch input.Kind {
	case Bet, Win, Loss, Refund, Rollback:
	default:
		return nil, ErrInvalidKind
	}

	if input.Money.Currency() == "" {
		return nil, money.ErrInvalidMoney
	}

	if input.Kind == Loss {
		if input.Money.Amount() != 0 {
			return nil, ErrInvalidAmount
		}
	} else if input.Money.Amount() <= 0 {
		return nil, ErrInvalidAmount
	}

	reference := input.ReferenceExternalTransactionID

	switch input.Kind {
	case Refund, Rollback:
		if strings.TrimSpace(reference) == "" {
			return nil, ErrInvalidReference
		}
	case Win:
		if reference != "" && strings.TrimSpace(reference) == "" {
			return nil, ErrInvalidReference
		}
	default:
		if reference != "" {
			return nil, ErrInvalidReference
		}
	}

	return &Transaction{
		id:                             input.ID,
		providerID:                     input.ProviderID,
		externalTransactionID:          input.ExternalTransactionID,
		idempotencyKey:                 input.IdempotencyKey,
		payloadHash:                    input.PayloadHash,
		walletID:                       input.WalletID,
		playerID:                       input.PlayerID,
		roundID:                        input.RoundID,
		gameID:                         input.GameID,
		kind:                           input.Kind,
		amount:                         input.Money,
		referenceExternalTransactionID: reference,
		status:                         Pending,
		origin:                         ExternalOrigin,
		createdAt:                      now.UTC(),
		updatedAt:                      now.UTC(),
	}, nil
}

func (t *Transaction) validateTransition(now time.Time) error {
	if t == nil || t.id == "" {
		return ErrInvalidTransaction
	}

	switch t.status {
	case Processed, Rejected, Failed:
		return ErrTerminalState
	case Pending, PendingReference:
	default:
		return ErrInvalidTransaction
	}

	if now.IsZero() || now.Before(t.createdAt) {
		return ErrInvalidTime
	}

	return nil
}

func (t *Transaction) AwaitReference(now time.Time) error {
	if err := t.validateTransition(now); err != nil {
		return err
	}

	if strings.TrimSpace(t.referenceExternalTransactionID) == "" {
		return ErrInvalidReference
	}

	t.status = PendingReference
	t.updatedAt = now.UTC()

	return nil
}

func (t *Transaction) Process(
	balance money.Money,
	referenceTransactionID string,
	now time.Time,
) error {
	if err := t.validateTransition(now); err != nil {
		return err
	}

	if _, err := t.amount.Compare(balance); err != nil {
		return err
	}

	if balance.Amount() < 0 {
		return ErrInvalidAmount
	}

	if t.referenceExternalTransactionID != "" {
		if strings.TrimSpace(referenceTransactionID) == "" {
			return ErrInvalidReference
		}
	} else if referenceTransactionID != "" {
		return ErrInvalidReference
	}

	t.status = Processed
	t.resultBalance = balance
	t.referenceTransactionID = referenceTransactionID
	t.processedAt = now.UTC()
	t.updatedAt = now.UTC()

	return nil
}

func (t *Transaction) Reject(failureCode string, now time.Time) error {
	if err := t.validateTransition(now); err != nil {
		return err
	}

	if strings.TrimSpace(failureCode) == "" {
		return ErrInvalidFailureCode
	}

	t.status = Rejected
	t.failureCode = failureCode
	t.updatedAt = now.UTC()

	return nil
}

func (t *Transaction) Fail(failureCode string, now time.Time) error {
	if err := t.validateTransition(now); err != nil {
		return err
	}

	if strings.TrimSpace(failureCode) == "" {
		return ErrInvalidFailureCode
	}

	t.status = Failed
	t.failureCode = failureCode
	t.updatedAt = now.UTC()

	return nil
}

func NewOpening(input OpeningInput, now time.Time) (*Transaction, error) {
	if strings.TrimSpace(input.ID) == "" ||
		strings.TrimSpace(input.WalletID) == "" ||
		strings.TrimSpace(input.PlayerID) == "" {
		return nil, ErrInvalidIdentity
	}

	if now.IsZero() {
		return nil, ErrInvalidTime
	}

	if input.Money.Currency() == "" {
		return nil, money.ErrInvalidMoney
	}

	if input.Money.Amount() <= 0 {
		return nil, ErrInvalidAmount
	}

	return &Transaction{
		id:        input.ID,
		origin:    InternalOrigin,
		walletID:  input.WalletID,
		playerID:  input.PlayerID,
		kind:      Opening,
		amount:    input.Money,
		status:    Pending,
		createdAt: now.UTC(),
		updatedAt: now.UTC(),
	}, nil
}

func (t *Transaction) ID() string {
	return t.id
}

func (t *Transaction) Kind() Kind {
	return t.kind
}

func (t *Transaction) Status() Status {
	return t.status
}

func (t *Transaction) Money() money.Money {
	return t.amount
}

func (t *Transaction) CreatedAt() time.Time {
	return t.createdAt
}

func (t *Transaction) UpdatedAt() time.Time {
	return t.updatedAt
}

func (t *Transaction) FailureCode() string {
	return t.failureCode
}

func (t *Transaction) ResultBalance() (money.Money, bool) {
	return t.resultBalance, t.status == Processed
}

func (t *Transaction) ProcessedAt() time.Time {
	return t.processedAt
}

func (t *Transaction) ReferenceTransactionID() string {
	return t.referenceTransactionID
}

func (t *Transaction) Origin() Origin {
	return t.origin
}

func (t *Transaction) WalletID() string {
	return t.walletID
}

func (t *Transaction) PlayerID() string {
	return t.playerID
}

func (t *Transaction) ProviderID() string {
	return t.providerID
}

func (t *Transaction) ExternalTransactionID() string {
	return t.externalTransactionID
}

func (t *Transaction) IdempotencyKey() string {
	return t.idempotencyKey
}

func (t *Transaction) PayloadHash() string {
	return t.payloadHash
}

func (t *Transaction) RoundID() string {
	return t.roundID
}

func (t *Transaction) GameID() string {
	return t.gameID
}

func (t *Transaction) ReferenceExternalTransactionID() string {
	return t.referenceExternalTransactionID
}
