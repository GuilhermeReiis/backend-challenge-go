package wager

import (
	"time"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
)

type Snapshot struct {
	ID                             string
	Origin                         Origin
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
	ReferenceTransactionID         string
	Status                         Status
	FailureCode                    string
	ResultBalance                  money.Money
	ProcessedAt                    time.Time
	CreatedAt                      time.Time
	UpdatedAt                      time.Time
}

func Rehydrate(state Snapshot) (*Transaction, error) {
	err := validateSnapshot(&state)
	if err != nil {
		return nil, err
	}

	return &Transaction{
		id:                             state.ID,
		origin:                         state.Origin,
		providerID:                     state.ProviderID,
		externalTransactionID:          state.ExternalTransactionID,
		idempotencyKey:                 state.IdempotencyKey,
		payloadHash:                    state.PayloadHash,
		walletID:                       state.WalletID,
		playerID:                       state.PlayerID,
		roundID:                        state.RoundID,
		gameID:                         state.GameID,
		kind:                           state.Kind,
		amount:                         state.Money,
		referenceExternalTransactionID: state.ReferenceExternalTransactionID,
		referenceTransactionID:         state.ReferenceTransactionID,
		status:                         state.Status,
		failureCode:                    state.FailureCode,
		resultBalance:                  state.ResultBalance,
		processedAt:                    state.ProcessedAt.UTC(),
		createdAt:                      state.CreatedAt.UTC(),
		updatedAt:                      state.UpdatedAt.UTC(),
	}, nil
}
