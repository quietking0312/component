package msql

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/jmoiron/sqlx"
)

// SqlxDB wraps sqlx.DB with configured timeouts and helper methods.
type SqlxDB struct {
	DB     *sqlx.DB
	Config *Config
}

// NewSqlxDB creates a new SqlxDB instance backed by a standard DB.
// Callers must import their database driver beforehand.
func NewSqlxDB(opts ...Option) (*SqlxDB, error) {
	db, err := NewDB(opts...)
	if err != nil {
		return nil, err
	}
	return &SqlxDB{
		DB:     sqlx.NewDb(db.DB, db.Config.DriverName),
		Config: db.Config,
	}, nil
}

// Close closes the underlying database connection.
func (s *SqlxDB) Close() error {
	return s.DB.Close()
}

// BeginTx starts a transaction with an optional isolation level and executes
// callbacks. If any callback returns an error, the transaction is rolled back.
func (s *SqlxDB) BeginTx(opts TxOption, cbs ...func(tx *sqlx.Tx, ctx context.Context) error) error {
	txOpts := DefaultTxOptions()
	if opts != nil {
		opts(txOpts)
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.Config.QueryTimeout)
	defer cancel()

	tx, err := s.DB.BeginTxx(ctx, txOpts)
	if err != nil {
		return err
	}

	for _, cb := range cbs {
		if err := cb(tx, ctx); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}
	return nil
}

// BeginContext carries transaction state and after-commit hooks.
type BeginContext struct {
	tx      *sqlx.Tx
	ctx     context.Context
	afterFc []func()
}

// Tx returns the current transaction.
func (b *BeginContext) Tx() *sqlx.Tx {
	return b.tx
}

// Ctx returns the transaction context.
func (b *BeginContext) Ctx() context.Context {
	return b.ctx
}

// AppendAfter registers a function to run after the transaction commits successfully.
func (b *BeginContext) AppendAfter(fc func()) {
	b.afterFc = append(b.afterFc, fc)
}

// BeginTxWithAfter starts a transaction with after-commit hooks.
// After-hooks run only if the transaction commits successfully.
func (s *SqlxDB) BeginTxWithAfter(opts TxOption, cbs ...func(ctx *BeginContext) error) error {
	txOpts := DefaultTxOptions()
	if opts != nil {
		opts(txOpts)
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.Config.QueryTimeout)
	defer cancel()

	tx, err := s.DB.BeginTxx(ctx, txOpts)
	if err != nil {
		return err
	}

	begCtx := &BeginContext{
		tx:      tx,
		ctx:     ctx,
		afterFc: make([]func(), 0),
	}

	for _, cb := range cbs {
		if err := cb(begCtx); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	for _, cb := range begCtx.afterFc {
		cb()
	}
	return nil
}

// NamedExec executes a named query and returns the result.
func (s *SqlxDB) NamedExec(query string, arg interface{}) (sql.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.Config.QueryTimeout)
	defer cancel()
	return s.DB.NamedExecContext(ctx, query, arg)
}

// Exec executes a query and returns the result.
func (s *SqlxDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.Config.QueryTimeout)
	defer cancel()
	return s.DB.ExecContext(ctx, query, args...)
}

// NamedQuery executes a named query and passes the rows to the callback.
// The rows are automatically closed after the callback returns.
func (s *SqlxDB) NamedQuery(query string, args interface{}, cb func(rows *sqlx.Rows) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.Config.QueryTimeout)
	defer cancel()

	rows, err := s.DB.NamedQueryContext(ctx, query, args)
	if err != nil {
		return err
	}
	defer rows.Close()

	return cb(rows)
}

// QueryRow executes a query that returns a single row and passes it to the callback.
func (s *SqlxDB) QueryRow(query string, args []interface{}, cb func(row *sqlx.Row) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.Config.QueryTimeout)
	defer cancel()

	row := s.DB.QueryRowxContext(ctx, query, args...)
	return cb(row)
}

// Query executes a query and passes the rows to the callback.
// The rows are automatically closed after the callback returns.
func (s *SqlxDB) Query(query string, args []interface{}, cb func(rows *sqlx.Rows) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.Config.QueryTimeout)
	defer cancel()

	rows, err := s.DB.QueryxContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	return cb(rows)
}

// Get executes a query and scans the single result into dest.
func (s *SqlxDB) Get(dest interface{}, query string, args ...interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.Config.QueryTimeout)
	defer cancel()
	return s.DB.GetContext(ctx, dest, query, args...)
}

// Select executes a query and scans all results into dest.
func (s *SqlxDB) Select(dest interface{}, query string, args ...interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.Config.QueryTimeout)
	defer cancel()
	return s.DB.SelectContext(ctx, dest, query, args...)
}

// In expands slice arguments in a query. The args must contain at least one slice or array.
func (s *SqlxDB) In(query string, args ...interface{}) (string, []interface{}, error) {
	if len(args) == 0 {
		return "", nil, fmt.Errorf("msql.In: at least one argument is required")
	}
	for _, arg := range args {
		k := reflect.TypeOf(arg).Kind()
		if k == reflect.Slice || k == reflect.Array {
			return sqlx.In(query, args...)
		}
	}
	return "", nil, fmt.Errorf("msql.In: at least one argument must be a slice or array")
}
