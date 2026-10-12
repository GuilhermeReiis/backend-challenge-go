package mapper

import (
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/ledger"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
	models "github.com/GuilhermeReiis/backend-challenge-go/internal/repository/postgres/model"
)

func LedgerFromModel(model models.Ledger) (ledger.Entry, error) {
	amount, err := money.New(model.AmountMinor, model.Currency)
	if err != nil {
		return ledger.Entry{}, err
	}
	before, err := money.New(model.BalanceBeforeMinor, model.Currency)
	if err != nil {
		return ledger.Entry{}, err
	}
	after, err := money.New(model.BalanceAfterMinor, model.Currency)
	if err != nil {
		return ledger.Entry{}, err
	}
	return ledger.New(model.ID, model.WalletID, model.TransactionID, ledger.Direction(model.Direction), amount, before, after, model.CreatedAt)
}

func LedgerToModel(entry ledger.Entry) (models.Ledger, error) {
	model := models.Ledger{
		ID: entry.ID(), WalletID: entry.WalletID(), TransactionID: entry.TransactionID(), TransactionStatus: "PROCESSED",
		Direction: string(entry.Direction()), Currency: entry.Money().Currency(), AmountMinor: entry.Money().Amount(),
		BalanceBeforeMinor: entry.BalanceBefore().Amount(), BalanceAfterMinor: entry.BalanceAfter().Amount(), CreatedAt: databaseTime(entry.CreatedAt()),
	}
	_, err := LedgerFromModel(model)
	if err != nil {
		return models.Ledger{}, err
	}
	return model, nil
}
