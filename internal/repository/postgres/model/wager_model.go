package model

import "time"

type Wager struct {
	ID                             string     `gorm:"column:id;primaryKey"`
	Origin                         string     `gorm:"column:origin"`
	Kind                           string     `gorm:"column:kind"`
	Status                         string     `gorm:"column:status"`
	WalletID                       string     `gorm:"column:wallet_id"`
	PlayerID                       string     `gorm:"column:player_id"`
	Currency                       string     `gorm:"column:currency"`
	AmountMinor                    int64      `gorm:"column:amount_minor"`
	ProviderID                     *string    `gorm:"column:provider_id"`
	ExternalTransactionID          *string    `gorm:"column:external_transaction_id"`
	IdempotencyKey                 *string    `gorm:"column:idempotency_key"`
	PayloadHash                    *string    `gorm:"column:payload_hash"`
	RoundID                        *string    `gorm:"column:round_id"`
	GameID                         *string    `gorm:"column:game_id"`
	ReferenceExternalTransactionID *string    `gorm:"column:reference_external_transaction_id"`
	ReferenceTransactionID         *string    `gorm:"column:reference_transaction_id"`
	FailureCode                    *string    `gorm:"column:failure_code"`
	ResultBalanceMinor             *int64     `gorm:"column:result_balance_minor"`
	ProcessedAt                    *time.Time `gorm:"column:processed_at"`
	Attempts                       int        `gorm:"column:attempts"`
	NextAttemptAt                  *time.Time `gorm:"column:next_attempt_at"`
	ExpiresAt                      *time.Time `gorm:"column:expires_at"`
	CreatedAt                      time.Time  `gorm:"column:created_at;autoCreateTime:false"`
	UpdatedAt                      time.Time  `gorm:"column:updated_at;autoUpdateTime:false"`
}

func (Wager) TableName() string { return "wager_transactions" }
