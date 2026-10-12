package ledger

import (
	"time"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/domain/money"
)

type Entry struct {
	id            string
	walletID      string
	transactionID string
	direction     Direction
	amount        money.Money
	balanceBefore money.Money
	balanceAfter  money.Money
	createdAt     time.Time
}

func (e Entry) ID() string {
	return e.id
}

func (e Entry) WalletID() string {
	return e.walletID
}

func (e Entry) TransactionID() string {
	return e.transactionID
}

func (e Entry) Direction() Direction {
	return e.direction
}

func (e Entry) Money() money.Money {
	return e.amount
}

func (e Entry) BalanceBefore() money.Money {
	return e.balanceBefore
}

func (e Entry) BalanceAfter() money.Money {
	return e.balanceAfter
}

func (e Entry) CreatedAt() time.Time {
	return e.createdAt
}
