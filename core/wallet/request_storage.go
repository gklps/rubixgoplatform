package wallet

import "time"

const RequestsTable = "requests"

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

// CreateRequest inserts a new request record
func (w *Wallet) CreateRequest(req *RequestRecord) error {
	return w.s.Write(RequestsTable, req)
}

// UpdateRequestStatus updates the status and transaction ID of a request
func (w *Wallet) UpdateRequestStatus(requestID, status, transactionID string) error {
	rec := &RequestRecord{TransactionID: transactionID, Status: status}
	return w.s.Update(RequestsTable, rec, "request_id=?", requestID)
}

// GetRequest retrieves a request by ID
func (w *Wallet) GetRequest(requestID string, req *RequestRecord) error {
	return w.s.Read(RequestsTable, req, "request_id=?", requestID)
}
