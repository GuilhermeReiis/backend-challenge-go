package event

import (
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/wager"
)

type RejectedData struct {
	TransactionID         string      `json:"transactionId"`
	WalletID              string      `json:"walletId"`
	PlayerID              string      `json:"playerId"`
	ProviderID            string      `json:"providerId,omitempty"`
	ExternalTransactionID string      `json:"externalTransactionId,omitempty"`
	Kind                  wager.Kind  `json:"kind"`
	Money                 money.Money `json:"money"`
	FailureCode           string      `json:"failureCode"`
}

func NewRejected(
	metadata Metadata,
	transaction *wager.Transaction,
) (Envelope[RejectedData], error) {
	if transaction == nil || transaction.Status() != wager.Rejected {
		return Envelope[RejectedData]{}, ErrInvalidData
	}

	data := RejectedData{
		TransactionID:         transaction.ID(),
		WalletID:              transaction.WalletID(),
		PlayerID:              transaction.PlayerID(),
		ProviderID:            transaction.ProviderID(),
		ExternalTransactionID: transaction.ExternalTransactionID(),
		Kind:                  transaction.Kind(),
		Money:                 transaction.Money(),
		FailureCode:           transaction.FailureCode(),
	}

	return newEnvelope(
		metadata,
		"WagerTransactionRejected",
		transaction.WalletID(),
		data,
	)
}
