package model

import "time"

type Inbox struct {
	ConsumerName string     `gorm:"column:consumer_name;primaryKey"`
	MessageID    string     `gorm:"column:message_id;primaryKey"`
	PayloadHash  string     `gorm:"column:payload_hash"`
	ReceivedAt   time.Time  `gorm:"column:received_at"`
	CompletedAt  *time.Time `gorm:"column:completed_at"`
}

func (Inbox) TableName() string { return "inbox_messages" }
