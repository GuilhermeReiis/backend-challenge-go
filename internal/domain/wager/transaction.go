package wager

import (
	"strings"
	"time"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
)

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

	if now.IsZero() || now.Before(t.updatedAt) {
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
