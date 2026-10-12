package event

import (
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/ledger"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
)

type BalanceChangedData struct {
	WalletID      string           `json:"walletId"`
	TransactionID string           `json:"transactionId"`
	Direction     ledger.Direction `json:"direction"`
	Money         money.Money      `json:"money"`
	BalanceBefore money.Money      `json:"balanceBefore"`
	BalanceAfter  money.Money      `json:"balanceAfter"`
	WalletVersion int64            `json:"walletVersion"`
}

func NewBalanceChanged(
	metadata Metadata,
	entry ledger.Entry,
	walletVersion int64,
) (Envelope[BalanceChangedData], error) {
	if entry.ID() == "" || walletVersion < 1 {
		return Envelope[BalanceChangedData]{}, ErrInvalidData
	}

	data := BalanceChangedData{
		WalletID:      entry.WalletID(),
		TransactionID: entry.TransactionID(),
		Direction:     entry.Direction(),
		Money:         entry.Money(),
		BalanceBefore: entry.BalanceBefore(),
		BalanceAfter:  entry.BalanceAfter(),
		WalletVersion: walletVersion,
	}

	return newEnvelope(
		metadata,
		"WalletBalanceChanged",
		entry.WalletID(),
		data,
	)
}
