package msql

import (
	"database/sql"
)

// DB wraps the standard library sql.DB.
type DB struct {
	DB     *sql.DB
	Config *Config
}

// NewDB opens a database connection, configures the pool, and verifies
// connectivity with Ping. Callers must import their database driver beforehand.
//
// Example:
//
//	import _ "github.com/go-sql-driver/mysql"
//	db, err := msql.NewDB(msql.DriverName("mysql"), msql.DataSourceName(dsn))
func NewDB(opts ...Option) (*DB, error) {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	db, err := sql.Open(cfg.DriverName, cfg.DataSourceName)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &DB{
		DB:     db,
		Config: cfg,
	}, nil
}

// Ping verifies a connection to the database is still alive.
func (db *DB) Ping() error {
	return db.DB.Ping()
}

// Close closes the database and prevents new queries from starting.
func (db *DB) Close() error {
	return db.DB.Close()
}
