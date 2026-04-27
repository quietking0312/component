package msql

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// newTestDB creates an in-memory SqlxDB for testing.
// MaxOpenConns is set to 1 so that :memory: is shared across queries.
func newTestDB(t *testing.T) *SqlxDB {
	db, err := NewSqlxDB(
		DriverName("sqlite"),
		DataSourceName(":memory:"),
		MaxOpenConns(1),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, 5, cfg.MaxIdleConns)
	assert.Equal(t, 25, cfg.MaxOpenConns)
	assert.Equal(t, 3*time.Second, cfg.QueryTimeout)
	assert.Equal(t, 30*time.Minute, cfg.ConnMaxIdleTime)
	assert.Equal(t, time.Hour, cfg.ConnMaxLifetime)
}

func TestNewDB(t *testing.T) {
	db, err := NewDB(
		DriverName("sqlite"),
		DataSourceName(":memory:"),
	)
	require.NoError(t, err)
	require.NotNil(t, db)

	assert.NoError(t, db.Ping())
	assert.NoError(t, db.Close())
}

func TestNewDB_InvalidDSN(t *testing.T) {
	_, err := NewDB(
		DriverName("sqlite"),
		DataSourceName("/this/path/cannot/exist/for/sure/test.db"),
	)
	require.Error(t, err)
}

func TestExec(t *testing.T) {
	db := newTestDB(t)

	_, err := db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)`)
	require.NoError(t, err)

	res, err := db.Exec(`INSERT INTO users (name, age) VALUES (?, ?)`, "alice", 30)
	require.NoError(t, err)

	affected, err := res.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected)
}

func TestGet(t *testing.T) {
	db := newTestDB(t)
	_, err := db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO users (name, age) VALUES (?, ?)`, "bob", 25)
	require.NoError(t, err)

	type user struct {
		ID   int    `db:"id"`
		Name string `db:"name"`
		Age  int    `db:"age"`
	}

	var u user
	err = db.Get(&u, `SELECT id, name, age FROM users WHERE name = ?`, "bob")
	require.NoError(t, err)
	assert.Equal(t, "bob", u.Name)
	assert.Equal(t, 25, u.Age)
}

func TestSelect(t *testing.T) {
	db := newTestDB(t)
	_, err := db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO users (name, age) VALUES (?, ?), (?, ?)`, "a", 1, "b", 2)
	require.NoError(t, err)

	type user struct {
		Name string `db:"name"`
		Age  int    `db:"age"`
	}

	var users []user
	err = db.Select(&users, `SELECT name, age FROM users ORDER BY age`)
	require.NoError(t, err)
	require.Len(t, users, 2)
	assert.Equal(t, "a", users[0].Name)
	assert.Equal(t, "b", users[1].Name)
}

func TestNamedExec(t *testing.T) {
	db := newTestDB(t)
	_, err := db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)`)
	require.NoError(t, err)

	res, err := db.NamedExec(
		`INSERT INTO users (name, age) VALUES (:name, :age)`,
		map[string]interface{}{"name": "charlie", "age": 35},
	)
	require.NoError(t, err)

	affected, _ := res.RowsAffected()
	assert.Equal(t, int64(1), affected)
}

func TestNamedQuery(t *testing.T) {
	db := newTestDB(t)
	_, err := db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO users (name, age) VALUES (?, ?)`, "dave", 40)
	require.NoError(t, err)

	type user struct {
		Name string `db:"name"`
	}

	err = db.NamedQuery(`SELECT name FROM users WHERE age = :age`, map[string]interface{}{"age": 40},
		func(rows *sqlx.Rows) error {
			require.True(t, rows.Next())
			var u user
			require.NoError(t, rows.StructScan(&u))
			assert.Equal(t, "dave", u.Name)
			return nil
		},
	)
	require.NoError(t, err)
}

func TestQueryRow(t *testing.T) {
	db := newTestDB(t)
	_, err := db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO users (name, age) VALUES (?, ?)`, "eve", 28)
	require.NoError(t, err)

	err = db.QueryRow(`SELECT name FROM users WHERE age = ?`, []interface{}{28},
		func(row *sqlx.Row) error {
			var name string
			require.NoError(t, row.Scan(&name))
			assert.Equal(t, "eve", name)
			return nil
		},
	)
	require.NoError(t, err)
}

func TestQuery(t *testing.T) {
	db := newTestDB(t)
	_, err := db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO users (name) VALUES (?), (?)`, "x", "y")
	require.NoError(t, err)

	var names []string
	err = db.Query(`SELECT name FROM users ORDER BY name`, []interface{}{},
		func(rows *sqlx.Rows) error {
			for rows.Next() {
				var n string
				require.NoError(t, rows.Scan(&n))
				names = append(names, n)
			}
			return nil
		},
	)
	require.NoError(t, err)
	assert.Equal(t, []string{"x", "y"}, names)
}

func TestBeginTx(t *testing.T) {
	db := newTestDB(t)
	_, err := db.Exec(`CREATE TABLE accounts (id INTEGER PRIMARY KEY, balance INTEGER)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO accounts (balance) VALUES (100)`)
	require.NoError(t, err)

	err = db.BeginTx(nil,
		func(tx *sqlx.Tx, ctx context.Context) error {
			_, err := tx.Exec(`UPDATE accounts SET balance = balance + 50 WHERE id = 1`)
			return err
		},
	)
	require.NoError(t, err)

	var balance int
	err = db.Get(&balance, `SELECT balance FROM accounts WHERE id = 1`)
	require.NoError(t, err)
	assert.Equal(t, 150, balance)
}

func TestBeginTx_Rollback(t *testing.T) {
	db := newTestDB(t)
	_, err := db.Exec(`CREATE TABLE accounts (id INTEGER PRIMARY KEY, balance INTEGER)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO accounts (balance) VALUES (100)`)
	require.NoError(t, err)

	expectedErr := errors.New("simulated failure")
	err = db.BeginTx(nil,
		func(tx *sqlx.Tx, ctx context.Context) error {
			_, err := tx.Exec(`UPDATE accounts SET balance = 999 WHERE id = 1`)
			require.NoError(t, err)
			return expectedErr
		},
	)
	require.ErrorIs(t, err, expectedErr)

	var balance int
	err = db.Get(&balance, `SELECT balance FROM accounts WHERE id = 1`)
	require.NoError(t, err)
	assert.Equal(t, 100, balance) // rollback succeeded
}

func TestBeginTxWithAfter(t *testing.T) {
	db := newTestDB(t)
	_, err := db.Exec(`CREATE TABLE logs (id INTEGER PRIMARY KEY, msg TEXT)`)
	require.NoError(t, err)

	var afterCalled bool
	err = db.BeginTxWithAfter(nil,
		func(ctx *BeginContext) error {
			_, err := ctx.Tx().Exec(`INSERT INTO logs (msg) VALUES (?)`, "tx done")
			require.NoError(t, err)

			ctx.AppendAfter(func() {
				afterCalled = true
			})
			return nil
		},
	)
	require.NoError(t, err)
	assert.True(t, afterCalled)

	var count int
	err = db.Get(&count, `SELECT COUNT(*) FROM logs`)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestIn(t *testing.T) {
	db := newTestDB(t)
	_, err := db.Exec(`CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO items (name) VALUES (?), (?), (?)`, "a", "b", "c")
	require.NoError(t, err)

	query, args, err := db.In(`SELECT name FROM items WHERE id IN (?)`, []int{1, 3})
	require.NoError(t, err)

	var names []string
	err = db.Select(&names, query, args...)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "c"}, names)
}

func TestIn_NoSlice(t *testing.T) {
	db := newTestDB(t)
	_, _, err := db.In(`SELECT 1`, 1, 2, 3)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be a slice or array")
}

func TestIn_EmptyArgs(t *testing.T) {
	db := newTestDB(t)
	_, _, err := db.In(`SELECT 1`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one argument")
}
