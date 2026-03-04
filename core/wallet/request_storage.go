package wallet

import "time"

// RequestRecord stores quorum/transfer request state
type RequestRecord struct {
	RequestID     string    `gorm:"column:request_id;primaryKey" json:"request_id"`
	TransactionID string    `gorm:"column:transaction_id" json:"transaction_id"`
	Status        string    `gorm:"column:status;not null" json:"status"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (RequestRecord) TableName() string {
	return "requests"
}
