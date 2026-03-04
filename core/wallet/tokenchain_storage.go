package wallet

// TokenChainEntry stores each step in a token's chain history
type TokenChainEntry struct {
	TokenID       string `gorm:"column:token_id;not null;uniqueIndex:idx_tokenchain_token_pos" json:"token_id"`
	TransactionID string `gorm:"column:transaction_id;not null" json:"transaction_id"`
	Role          int16  `gorm:"column:role;not null" json:"role"`
	Position      int64  `gorm:"column:position;not null;uniqueIndex:idx_tokenchain_token_pos" json:"position"`
}

func (TokenChainEntry) TableName() string {
	return "tokenchain"
}
