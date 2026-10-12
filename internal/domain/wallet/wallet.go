package wallet

import (
	"math"
	"time"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
)

type Wallet struct {
	id        string
	playerID  string
	balance   money.Money
	version   int64
	createdAt time.Time
	updatedAt time.Time
}

func (w *Wallet) ID() string {
	return w.id
}

func (w *Wallet) PlayerID() string {
	return w.playerID
}

func (w *Wallet) Currency() string {
	return w.balance.Currency()
}

func (w *Wallet) Balance() money.Money {
	return w.balance
}

func (w *Wallet) Version() int64 {
	return w.version
}

func (w *Wallet) CreatedAt() time.Time {
	return w.createdAt
}

func (w *Wallet) UpdatedAt() time.Time {
	return w.updatedAt
}

func (w *Wallet) Credit(amount money.Money, now time.Time) error {
	if err := w.validateMovement(amount, now); err != nil {
		return err
	}

	if amount.Amount() == 0 {
		return nil
	}

	if w.version == math.MaxInt64 {
		return ErrVersionOverflow
	}

	newBalance, err := w.balance.Add(amount)

	if err != nil {
		return err
	}

	w.balance = newBalance
	w.version++
	w.updatedAt = now.UTC()

	return nil
}

func (w *Wallet) Debit(amount money.Money, now time.Time) error {
	if err := w.validateMovement(amount, now); err != nil {
		return err
	}

	if amount.Amount() == 0 {
		return nil
	}

	comparison, err := w.balance.Compare(amount)

	if err != nil {
		return err
	}

	if comparison < 0 {
		return ErrInsufficientFunds
	}

	if w.version == math.MaxInt64 {
		return ErrVersionOverflow
	}

	newBalance, err := w.balance.Sub(amount)

	if err != nil {
		return err
	}

	w.balance = newBalance
	w.version++
	w.updatedAt = now.UTC()

	return nil
}
