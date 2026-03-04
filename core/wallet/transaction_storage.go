package wallet

import "time"

// TransactionRecord stores details of a completed transaction
type TransactionRecord struct {
	ID        string    `gorm:"column:id;primaryKey" json:"id"`
	Info      string    `gorm:"column:info;type:text" json:"info"`
	Signature string    `gorm:"column:signature;type:text" json:"signature"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (TransactionRecord) TableName() string {
	return "transactions"
}
