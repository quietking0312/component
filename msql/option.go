package msql

import (
	"database/sql"
	"time"
)

// Config holds database connection configuration.
type Config struct {
	DriverName      string
	DataSourceName  string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
	QueryTimeout    time.Duration
}

// Option configures a Config.
type Option func(cfg *Config)

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		MaxIdleConns:    5,
		MaxOpenConns:    25,
		ConnMaxIdleTime: 30 * time.Minute,
		ConnMaxLifetime: 1 * time.Hour,
		QueryTimeout:    3 * time.Second,
	}
}

// DriverName sets the database driver name (e.g. "mysql", "sqlite3", "postgres").
func DriverName(name string) Option {
	return func(cfg *Config) {
		cfg.DriverName = name
	}
}

// DataSourceName sets the DSN / connection string.
func DataSourceName(dsn string) Option {
	return func(cfg *Config) {
		cfg.DataSourceName = dsn
	}
}

// MaxIdleConns sets the maximum number of idle connections.
func MaxIdleConns(n int) Option {
	return func(cfg *Config) {
		cfg.MaxIdleConns = n
	}
}

// MaxOpenConns sets the maximum number of open connections.
func MaxOpenConns(n int) Option {
	return func(cfg *Config) {
		cfg.MaxOpenConns = n
	}
}

// ConnMaxIdleTime sets the maximum amount of time a connection may be idle.
func ConnMaxIdleTime(d time.Duration) Option {
	return func(cfg *Config) {
		cfg.ConnMaxIdleTime = d
	}
}

// ConnMaxLifetime sets the maximum amount of time a connection may be reused.
func ConnMaxLifetime(d time.Duration) Option {
	return func(cfg *Config) {
		cfg.ConnMaxLifetime = d
	}
}

// QueryTimeout sets the default query / transaction timeout.
func QueryTimeout(d time.Duration) Option {
	return func(cfg *Config) {
		cfg.QueryTimeout = d
	}
}

// TxOption customizes transaction options.
type TxOption func(options *sql.TxOptions)

// LevelReadCommitted uses the read-committed isolation level.
func LevelReadCommitted() TxOption {
	return func(options *sql.TxOptions) {
		options.Isolation = sql.LevelReadCommitted
	}
}

// LevelRepeatableRead uses the repeatable-read isolation level.
func LevelRepeatableRead() TxOption {
	return func(options *sql.TxOptions) {
		options.Isolation = sql.LevelRepeatableRead
	}
}

// LevelSerializable uses the serializable isolation level.
func LevelSerializable() TxOption {
	return func(options *sql.TxOptions) {
		options.Isolation = sql.LevelSerializable
	}
}

// DefaultTxOptions returns the default transaction options.
func DefaultTxOptions() *sql.TxOptions {
	return &sql.TxOptions{
		Isolation: sql.LevelDefault,
	}
}
