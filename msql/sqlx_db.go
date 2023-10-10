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

func (_sqlxDB *SqlxDB) SqlxBeginTx(opts TxOption, cbs ...func(tx *sqlx.Tx, ctx context.Context) error) error {
	defaultOpt := DefaultTxOptions()
	if opts != nil {
		opts(defaultOpt)
	}

	ctx, cancel := context.WithTimeout(context.Background(), _sqlxDB.DBCfg.MaxQueryTime)
	defer cancel()
	tx, err := _sqlxDB.SqlxDB.BeginTxx(ctx, defaultOpt)
	if err != nil {
		return err
	}
	for _, cb := range cbs {
		if err := cb(tx, ctx); err != nil {
			if err := tx.Rollback(); err != nil {
				return err
			}
			return err
		}
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
	return _sqlxDB.SqlxDB.NamedExecContext(ctx, format, arg)
}

func (_sqlxDB *SqlxDB) SqlxExec(format string, args ...interface{}) (sql.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), _sqlxDB.DBCfg.MaxQueryTime)
	defer cancel()
	return _sqlxDB.SqlxDB.ExecContext(ctx, format, args...)
}

func (_sqlxDB *SqlxDB) SqlxNameQuery(format string, args interface{}, cb func(rows *sqlx.Rows) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), _sqlxDB.DBCfg.MaxQueryTime)
	defer cancel()
	rows, err := _sqlxDB.SqlxDB.NamedQueryContext(ctx, format, args)
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
	row := _sqlxDB.SqlxDB.QueryRowxContext(ctx, format, args...)
	return cb(row)
}

func (_sqlxDB *SqlxDB) SqlxQuery(format string, args []interface{}, cb func(rows *sqlx.Rows) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), _sqlxDB.DBCfg.MaxQueryTime)
	defer cancel()
	rows, err := _sqlxDB.SqlxDB.QueryxContext(ctx, format, args...)
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
	return _sqlxDB.SqlxDB.GetContext(ctx, dest, format, args...)
}

func (_sqlxDB *SqlxDB) SqlxSelect(dest interface{}, format string, args ...interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), _sqlxDB.DBCfg.MaxQueryTime)
	defer cancel()
	return _sqlxDB.SqlxDB.SelectContext(ctx, dest, format, args...)
}

func (_sqlxDB *SqlxDB) In(format string, args ...interface{}) (string, []interface{}, error) {
	return sqlx.In(format, args...)
}
