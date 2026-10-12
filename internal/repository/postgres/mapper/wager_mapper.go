package mapper

import (
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/wager"
	models "github.com/GuilhermeReiis/backend-challenge-go/internal/repository/postgres/model"
)

func WagerFromModel(model models.Wager) (*wager.Transaction, error) {
	amount, err := money.New(model.AmountMinor, model.Currency)
	if err != nil {
		return nil, err
	}
	var result money.Money
	if model.ResultBalanceMinor != nil {
		result, err = money.New(*model.ResultBalanceMinor, model.Currency)
		if err != nil {
			return nil, err
		}
	}
	return wager.Rehydrate(wager.Snapshot{
		ID: model.ID, Origin: wager.Origin(model.Origin), Kind: wager.Kind(model.Kind), Status: wager.Status(model.Status),
		WalletID: model.WalletID, PlayerID: model.PlayerID, Money: amount,
		ProviderID: StringValue(model.ProviderID), ExternalTransactionID: StringValue(model.ExternalTransactionID),
		IdempotencyKey: StringValue(model.IdempotencyKey), PayloadHash: StringValue(model.PayloadHash),
		RoundID: StringValue(model.RoundID), GameID: StringValue(model.GameID),
		ReferenceExternalTransactionID: StringValue(model.ReferenceExternalTransactionID),
		ReferenceTransactionID:         StringValue(model.ReferenceTransactionID), FailureCode: StringValue(model.FailureCode),
		ResultBalance: result, ProcessedAt: timeValue(model.ProcessedAt), CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt,
	})
}

func WagerToModel(value *wager.Transaction) (models.Wager, error) {
	if value == nil {
		return models.Wager{}, wager.ErrInvalidTransaction
	}
	model := models.Wager{
		ID: value.ID(), Origin: string(value.Origin()), Kind: string(value.Kind()), Status: string(value.Status()),
		WalletID: value.WalletID(), PlayerID: value.PlayerID(), Currency: value.Money().Currency(), AmountMinor: value.Money().Amount(),
		ProviderID: optionalString(value.ProviderID()), ExternalTransactionID: optionalString(value.ExternalTransactionID()),
		IdempotencyKey: optionalString(value.IdempotencyKey()), PayloadHash: optionalString(value.PayloadHash()),
		RoundID: optionalString(value.RoundID()), GameID: optionalString(value.GameID()),
		ReferenceExternalTransactionID: optionalString(value.ReferenceExternalTransactionID()),
		ReferenceTransactionID:         optionalString(value.ReferenceTransactionID()), FailureCode: optionalString(value.FailureCode()),
		ProcessedAt: OptionalTime(value.ProcessedAt()), CreatedAt: databaseTime(value.CreatedAt()), UpdatedAt: databaseTime(value.UpdatedAt()),
	}
	result, available := value.ResultBalance()
	if available {
		minor := result.Amount()
		model.ResultBalanceMinor = &minor
	}
	_, err := WagerFromModel(model)
	if err != nil {
		return models.Wager{}, err
	}
	return model, nil
}
