package wallet

const TokenChainTable = "tokenchain"

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

// InsertTokenChainEntry inserts a new tokenchain entry
func (w *Wallet) InsertTokenChainEntry(entry *TokenChainEntry) error {
	return w.s.Write(TokenChainTable, entry)
}

// GetTokenChainHistory returns all tokenchain entries for a token, ordered by position ascending
func (w *Wallet) GetTokenChainHistory(tokenID string) ([]TokenChainEntry, error) {
	var entries []TokenChainEntry
	err := w.s.Read(TokenChainTable, &entries, "token_id=? ORDER BY position ASC", tokenID)
	return entries, err
}

// GetLatestPosition returns the maximum position for a token
func (w *Wallet) GetLatestPosition(tokenID string) (int64, error) {
	var entry TokenChainEntry
	err := w.s.Read(TokenChainTable, &entry, "token_id=? ORDER BY position DESC LIMIT 1", tokenID)
	if err != nil {
		return 0, err
	}
	return entry.Position, nil
}
