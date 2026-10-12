package wallet

import (
	"time"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
)

func New(
	id string,
	playerID string,
	initialBalance money.Money,
	now time.Time,
) (*Wallet, error) {
	err := validateState(id, playerID, initialBalance, 1, now, now)
	if err != nil {
		return nil, err
	}

	return &Wallet{
		id:        id,
		playerID:  playerID,
		balance:   initialBalance,
		version:   1,
		createdAt: now.UTC(),
		updatedAt: now.UTC(),
	}, nil
}
