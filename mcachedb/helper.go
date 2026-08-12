package mcachedb

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

// NewSQLXStoreFromDB 从现有 sqlx.DB 创建存储
func NewSQLXStoreFromDB(db *sqlx.DB, entityType Entity) (*SQLXStore, error) {
	config := DefaultSQLXStoreConfig()
	config.DSN = "" // 使用已有连接

	store := &SQLXStore{
		db:         db,
		config:     config,
		entityType: entityType,
	}

	if err := store.createTable(); err != nil {
		return nil, fmt.Errorf("create table failed: %w", err)
	}

	return store, nil
}
