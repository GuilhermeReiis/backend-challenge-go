package wallet

import (
	"time"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
)

func Rehydrate(
	id string,
	playerID string,
	balance money.Money,
	version int64,
	createdAt time.Time,
	updatedAt time.Time,
) (*Wallet, error) {
	err := validateState(
		id, playerID, balance, version, createdAt, updatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &Wallet{
		id:        id,
		playerID:  playerID,
		balance:   balance,
		version:   version,
		createdAt: createdAt.UTC(),
		updatedAt: updatedAt.UTC(),
	}, nil
}
