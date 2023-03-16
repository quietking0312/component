package msql

import (
	"context"
	"database/sql"
	"github.com/jmoiron/sqlx"
)

var (
	_sqlxDB *SqlxDB
)

type SqlxDB struct {
	SqlxDB *sqlx.DB
	DBCfg  *DBCfg
}

func NewSqlxDB(opts ...Option) (*SqlxDB, error) {
	db, err := NewDB(opts...)
	if err != nil {
		return nil, err
	}
	_sqlxDB = &SqlxDB{
		SqlxDB: sqlx.NewDb(db.DB, db.DBCfg.DriveName),
		DBCfg:  db.DBCfg,
	}
	return _sqlxDB, nil
}

func (_sqlxDB *SqlxDB) SqlxBeginTx(cb func(tx *sqlx.Tx, ctx context.Context) error, opts ...TxOption) error {
	defaultOpt := DefaultTxOptions()
	for _, opt := range opts {
		opt(defaultOpt)
	}
	ctx, cancel := context.WithTimeout(context.Background(), _sqlxDB.DBCfg.MaxQueryTime)
	defer cancel()
	tx, err := _sqlxDB.SqlxDB.BeginTxx(ctx, defaultOpt)
	if err != nil {
		return err
	}
	if err := cb(tx, ctx); err != nil {
		if err := tx.Rollback(); err != nil {
			return err
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		if err := tx.Rollback(); err != nil {
			return err
		}
		return err
	}
	return err
}

func (_sqlxDB *SqlxDB) SqlxNameExec(format string, arg interface{}) (sql.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), _sqlxDB.DBCfg.MaxQueryTime)
	defer cancel()
	namedStmt, err := _sqlxDB.SqlxDB.PrepareNamedContext(ctx, format)
	if err != nil {
		return nil, err
	}
	defer namedStmt.Close()
	return namedStmt.ExecContext(ctx, arg)
}

func (_sqlxDB *SqlxDB) SqlxExec(format string, args ...interface{}) (sql.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), _sqlxDB.DBCfg.MaxQueryTime)
	defer cancel()
	stmt, err := _sqlxDB.SqlxDB.PreparexContext(ctx, format)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	return stmt.ExecContext(ctx, args...)
}

func (_sqlxDB *SqlxDB) SqlxNameQuery(format string, args interface{}, cb func(rows *sqlx.Rows) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), _sqlxDB.DBCfg.MaxQueryTime)
	defer cancel()
	namedStmt, err := _sqlxDB.SqlxDB.PrepareNamedContext(ctx, format)
	if err != nil {
		return err
	}
	defer namedStmt.Close()
	rows, err := namedStmt.QueryxContext(ctx, args)
	if err != nil {
		return err
	}
	defer rows.Close()
	err = cb(rows)
	return err
}

func (_sqlxDB *SqlxDB) SqlxQueryRow(format string, args []interface{}, cb func(row *sqlx.Row) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), _sqlxDB.DBCfg.MaxQueryTime)
	defer cancel()
	stmt, err := _sqlxDB.SqlxDB.PreparexContext(ctx, format)
	if err != nil {
		return err
	}
	defer stmt.Close()
	row := stmt.QueryRowxContext(ctx, args...)
	return cb(row)
}

func (_sqlxDB *SqlxDB) SqlxQuery(format string, args []interface{}, cb func(rows *sqlx.Rows) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), _sqlxDB.DBCfg.MaxQueryTime)
	defer cancel()
	stmt, err := _sqlxDB.SqlxDB.PreparexContext(ctx, format)
	if err != nil {
		return err
	}
	defer stmt.Close()
	rows, err := stmt.QueryxContext(ctx, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	err = cb(rows)
	return err
}

func (_sqlxDB *SqlxDB) SqlxGet(dest interface{}, format string, args ...interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), _sqlxDB.DBCfg.MaxQueryTime)
	defer cancel()
	namedStmt, err := _sqlxDB.SqlxDB.PreparexContext(ctx, format)
	if err != nil {
		return err
	}
	defer namedStmt.Close()
	return namedStmt.GetContext(ctx, dest, args...)
}

func (_sqlxDB *SqlxDB) SqlxSelect(dest interface{}, format string, args ...interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), _sqlxDB.DBCfg.MaxQueryTime)
	defer cancel()
	namedStmt, err := _sqlxDB.SqlxDB.PreparexContext(ctx, format)
	if err != nil {
		return err
	}
	defer namedStmt.Close()
	return namedStmt.SelectContext(ctx, dest, args...)
}

func (_sqlxDB *SqlxDB) In(format string, args ...interface{}) (string, []interface{}, error) {
	return sqlx.In(format, args...)
}
