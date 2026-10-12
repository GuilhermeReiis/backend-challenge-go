package event

import (
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/wager"
)

type ProcessedData struct {
	TransactionID         string      `json:"transactionId"`
	WalletID              string      `json:"walletId"`
	PlayerID              string      `json:"playerId"`
	ProviderID            string      `json:"providerId,omitempty"`
	ExternalTransactionID string      `json:"externalTransactionId,omitempty"`
	Kind                  wager.Kind  `json:"kind"`
	Money                 money.Money `json:"money"`
	Balance               money.Money `json:"balance"`
}

func NewProcessed(
	metadata Metadata,
	transaction *wager.Transaction,
) (Envelope[ProcessedData], error) {
	if transaction == nil || transaction.Status() != wager.Processed {
		return Envelope[ProcessedData]{}, ErrInvalidData
	}

	balance, available := transaction.ResultBalance()

	if !available {
		return Envelope[ProcessedData]{}, ErrInvalidData
	}

	data := ProcessedData{
		TransactionID:         transaction.ID(),
		WalletID:              transaction.WalletID(),
		PlayerID:              transaction.PlayerID(),
		ProviderID:            transaction.ProviderID(),
		ExternalTransactionID: transaction.ExternalTransactionID(),
		Kind:                  transaction.Kind(),
		Money:                 transaction.Money(),
		Balance:               balance,
	}

	return newEnvelope(
		metadata,
		"WagerTransactionProcessed",
		transaction.WalletID(),
		data,
	)
}
