package wallet

import (
	"strings"
	"time"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
)

func validateState(
	id string,
	playerID string,
	balance money.Money,
	version int64,
	createdAt time.Time,
	updatedAt time.Time,
) error {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(playerID) == "" {
		return ErrInvalidIdentity
	}

	if version < 1 {
		return ErrInvalidVersion
	}

	if createdAt.IsZero() || updatedAt.IsZero() || updatedAt.Before(createdAt) {
		return ErrInvalidTime
	}

	if balance.Currency() == "" {
		return money.ErrInvalidMoney
	}

	zero, err := money.Zero(balance.Currency())

	if err != nil {
		return err
	}

	comparison, err := balance.Compare(zero)

	if err != nil {
		return err
	}

	if comparison < 0 {
		return ErrNegativeBalance
	}

	return nil
}

func (w *Wallet) validateMovement(amount money.Money, now time.Time) error {
	if w == nil {
		return ErrInvalidWallet
	}

	if err := validateState(
		w.id, w.playerID, w.balance, w.version, w.createdAt, w.updatedAt,
	); err != nil {
		return err
	}

	if now.IsZero() || now.Before(w.updatedAt) {
		return ErrInvalidTime
	}

	if _, err := w.balance.Compare(amount); err != nil {
		return err
	}

	if amount.Amount() < 0 {
		return ErrInvalidMovement
	}

	return nil
}
