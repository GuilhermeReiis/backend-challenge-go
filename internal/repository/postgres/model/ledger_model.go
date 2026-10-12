package model

import "time"

type Ledger struct {
	ID                 string    `gorm:"column:id;primaryKey"`
	Sequence           int64     `gorm:"column:sequence;->"`
	WalletID           string    `gorm:"column:wallet_id"`
	TransactionID      string    `gorm:"column:transaction_id"`
	TransactionStatus  string    `gorm:"column:transaction_status"`
	Direction          string    `gorm:"column:direction"`
	Currency           string    `gorm:"column:currency"`
	AmountMinor        int64     `gorm:"column:amount_minor"`
	BalanceBeforeMinor int64     `gorm:"column:balance_before_minor"`
	BalanceAfterMinor  int64     `gorm:"column:balance_after_minor"`
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime:false"`
}

func (Ledger) TableName() string { return "wallet_ledger_entries" }
