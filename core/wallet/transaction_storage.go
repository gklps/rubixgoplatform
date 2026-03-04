package wallet

import "time"

const TransactionsTable = "transactions"

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

// CreateTransaction inserts a new transaction record
func (w *Wallet) CreateTransaction(rec *TransactionRecord) error {
	return w.s.Write(TransactionsTable, rec)
}

// GetTransaction retrieves a transaction by ID
func (w *Wallet) GetTransaction(id string, rec *TransactionRecord) error {
	return w.s.Read(TransactionsTable, rec, "id=?", id)
}

// ListTransactionsByTokenID returns all transactions for a token (via tokenchain join)
func (w *Wallet) ListTransactionsByTokenID(tokenID string) ([]TransactionRecord, error) {
	var recs []TransactionRecord
	err := w.s.Read(TransactionsTable, &recs, "id IN (SELECT transaction_id FROM tokenchain WHERE token_id=?)", tokenID)
	return recs, err
}
