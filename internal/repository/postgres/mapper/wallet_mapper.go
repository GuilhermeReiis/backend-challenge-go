package mapper

import (
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/wallet"
	models "github.com/GuilhermeReiis/backend-challenge-go/internal/repository/postgres/model"
)

func WalletFromModel(model models.Wallet) (*wallet.Wallet, error) {
	balance, err := money.New(model.BalanceMinor, model.Currency)
	if err != nil {
		return nil, err
	}
	return wallet.Rehydrate(model.ID, model.PlayerID, balance, model.Version, model.CreatedAt, model.UpdatedAt)
}

func WalletToModel(value *wallet.Wallet) (models.Wallet, error) {
	if value == nil {
		return models.Wallet{}, wallet.ErrInvalidWallet
	}
	model := models.Wallet{
		ID: value.ID(), PlayerID: value.PlayerID(), Currency: value.Currency(),
		BalanceMinor: value.Balance().Amount(), Version: value.Version(),
		CreatedAt: databaseTime(value.CreatedAt()), UpdatedAt: databaseTime(value.UpdatedAt()),
	}
	_, err := WalletFromModel(model)
	if err != nil {
		return models.Wallet{}, err
	}
	return model, nil
}
