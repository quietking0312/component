package msql

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

var (
	_db *DB
)

type DB struct {
	DB    *sql.DB
	DBCfg *DBCfg
}

func NewDB(opts ...Option) (*DB, error) {
	cfg := DefaultDBOption()
	for _, opt := range opts {
		opt(cfg)
	}
	db, err := sql.Open(cfg.DriveName, cfg.DataSourceName)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(cfg.MaxOpenConnection)
	db.SetMaxIdleConns(cfg.MaxIdleConnection)
	db.SetConnMaxIdleTime(cfg.MaxQueryTime)
	_db = &DB{
		DB:    db,
		DBCfg: cfg,
	}
	return _db, err
}

func (db *DB) Ping() error {
	return db.DB.Ping()
}
