package wallet

import (
	"errors"
	"math"
	"strings"
	"time"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
)

var (
	ErrInvalidIdentity   = errors.New("wallet: invalid identity")
	ErrNegativeBalance   = errors.New("wallet: negative balance")
	ErrInvalidTime       = errors.New("wallet: invalid time")
	ErrInvalidVersion    = errors.New("wallet: invalid version")
	ErrInvalidWallet     = errors.New("wallet: uninitialized value")
	ErrInvalidMovement   = errors.New("wallet: negative movement")
	ErrInsufficientFunds = errors.New("wallet: insufficient funds")
	ErrVersionOverflow   = errors.New("wallet: version overflow")
)

type Wallet struct {
	id        string
	playerID  string
	balance   money.Money
	version   int64
	createdAt time.Time
	updatedAt time.Time
}

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

func New(
	id string,
	playerID string,
	initialBalance money.Money,
	now time.Time,
) (*Wallet, error) {

	if err := validateState(id, playerID, initialBalance, 1, now, now); err != nil {
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

func Rehydrate(
	id string,
	playerID string,
	balance money.Money,
	version int64,
	createdAt time.Time,
	updatedAt time.Time,
) (*Wallet, error) {
	if err := validateState(
		id, playerID, balance, version, createdAt, updatedAt,
	); err != nil {
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

func (w *Wallet) validateMovement(amount money.Money, now time.Time) error {
	if w == nil {
		return ErrInvalidWallet
	}

	if err := validateState(
		w.id, w.playerID, w.balance, w.version, w.createdAt, w.updatedAt,
	); err != nil {
		return err
	}

	if now.IsZero() || now.Before(w.createdAt) {
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
