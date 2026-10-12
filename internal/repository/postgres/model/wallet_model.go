package model

import "time"

type Wallet struct {
	ID           string    `gorm:"column:id;primaryKey"`
	PlayerID     string    `gorm:"column:player_id"`
	Currency     string    `gorm:"column:currency"`
	BalanceMinor int64     `gorm:"column:balance_minor"`
	Version      int64     `gorm:"column:version"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime:false"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime:false"`
}

func (Wallet) TableName() string { return "wallets" }
