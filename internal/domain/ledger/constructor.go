package ledger

import (
	"strings"
	"time"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
)

func New(
	id string,
	walletID string,
	transactionID string,
	direction Direction,
	amount money.Money,
	balanceBefore money.Money,
	balanceAfter money.Money,
	now time.Time,
) (Entry, error) {
	if strings.TrimSpace(id) == "" ||
		strings.TrimSpace(walletID) == "" ||
		strings.TrimSpace(transactionID) == "" {
		return Entry{}, ErrInvalidIdentity
	}

	if now.IsZero() {
		return Entry{}, ErrInvalidTime
	}

	if amount.Currency() == "" ||
		balanceBefore.Currency() == "" ||
		balanceAfter.Currency() == "" {
		return Entry{}, money.ErrInvalidMoney
	}

	if amount.Amount() <= 0 {
		return Entry{}, ErrInvalidAmount
	}

	if balanceBefore.Amount() < 0 || balanceAfter.Amount() < 0 {
		return Entry{}, ErrNegativeBalance
	}

	var expectedBalance money.Money
	var err error

	switch direction {
	case Debit:
		expectedBalance, err = balanceBefore.Sub(amount)
	case Credit:
		expectedBalance, err = balanceBefore.Add(amount)
	default:
		return Entry{}, ErrInvalidDirection
	}

	if err != nil {
		return Entry{}, err
	}

	comparison, err := expectedBalance.Compare(balanceAfter)
	if err != nil {
		return Entry{}, err
	}

	if comparison != 0 {
		return Entry{}, ErrInconsistentBalance
	}

	return Entry{
		id:            id,
		walletID:      walletID,
		transactionID: transactionID,
		direction:     direction,
		amount:        amount,
		balanceBefore: balanceBefore,
		balanceAfter:  balanceAfter,
		createdAt:     now.UTC(),
	}, nil
}
