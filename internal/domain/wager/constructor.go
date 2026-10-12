package wager

import (
	"strings"
	"time"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
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

	if err := validatePayloadHash(input.PayloadHash); err != nil {
		return nil, err
	}

	if !isValidExternalKind(input.Kind) {
		return nil, ErrInvalidKind
	}

	if err := validateAmount(input.Kind, input.Money); err != nil {
		return nil, err
	}

	reference := input.ReferenceExternalTransactionID

	if err := validateExternalReference(input.Kind, reference); err != nil {
		return nil, err
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

func NewOpening(input OpeningInput, now time.Time) (*Transaction, error) {
	if strings.TrimSpace(input.ID) == "" ||
		strings.TrimSpace(input.WalletID) == "" ||
		strings.TrimSpace(input.PlayerID) == "" {
		return nil, ErrInvalidIdentity
	}

	if now.IsZero() {
		return nil, ErrInvalidTime
	}

	if err := validateAmount(Opening, input.Money); err != nil {
		return nil, err
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
