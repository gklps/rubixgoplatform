package state

import (
	"time"

	"gorm.io/gorm"
)

// StateRootSnapshot stores a periodic snapshot of the network state root hash.
// Used for cross-node state comparison and corruption detection.
type StateRootSnapshot struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	BlockHeight int64     `gorm:"column:block_height;not null;index:idx_state_roots_height,sort:desc" json:"block_height"`
	StateRoot   string    `gorm:"column:state_root;not null;type:text" json:"state_root"`
	TokenCount  int64     `gorm:"column:token_count;not null" json:"token_count"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (StateRootSnapshot) TableName() string {
	return "state_roots"
}

// InitStateRoots creates the state_roots table and index if they don't exist.
func InitStateRoots(db *gorm.DB) error {
	if err := db.AutoMigrate(&StateRootSnapshot{}); err != nil {
		return err
	}
	return db.Exec(`CREATE INDEX IF NOT EXISTS idx_state_roots_height ON state_roots (block_height DESC)`).Error
}
