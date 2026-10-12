package event

import (
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/wager"
)

type PendingReferenceData struct {
	TransactionID                  string      `json:"transactionId"`
	WalletID                       string      `json:"walletId"`
	PlayerID                       string      `json:"playerId"`
	ProviderID                     string      `json:"providerId"`
	ExternalTransactionID          string      `json:"externalTransactionId"`
	Kind                           wager.Kind  `json:"kind"`
	Money                          money.Money `json:"money"`
	ReferenceExternalTransactionID string      `json:"referenceExternalTransactionId"`
}

func NewPendingReference(
	metadata Metadata,
	transaction *wager.Transaction,
) (Envelope[PendingReferenceData], error) {
	if transaction == nil || transaction.Status() != wager.PendingReference {
		return Envelope[PendingReferenceData]{}, ErrInvalidData
	}

	data := PendingReferenceData{
		TransactionID:                  transaction.ID(),
		WalletID:                       transaction.WalletID(),
		PlayerID:                       transaction.PlayerID(),
		ProviderID:                     transaction.ProviderID(),
		ExternalTransactionID:          transaction.ExternalTransactionID(),
		Kind:                           transaction.Kind(),
		Money:                          transaction.Money(),
		ReferenceExternalTransactionID: transaction.ReferenceExternalTransactionID(),
	}

	return newEnvelope(
		metadata,
		"WagerTransactionPendingReference",
		transaction.WalletID(),
		data,
	)
}
