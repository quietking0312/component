package mcachedb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// SQLX 层默认常量
const (
	defaultSQLXTableName     = "cachedb_entries"
	defaultSQLXKeyColumn     = "cache_key"
	defaultSQLXDataColumn    = "cache_data"
	defaultSQLXVerColumn     = "version"
	defaultSQLXDelColumn     = "is_deleted"
	defaultSQLXTimeColumn    = "updated_at"
	defaultSQLXMaxOpenConns  = 20
	defaultSQLXMaxIdleConns  = 5
	defaultSQLXMaxLifetime   = time.Hour
	defaultSQLXKeyVarcharLen = 255
)

// SQLXStoreConfig SQLX存储配置
type SQLXStoreConfig struct {
	DSN          string
	TableName    string
	KeyColumn    string
	DataColumn   string
	VerColumn    string
	DelColumn    string
	TimeColumn   string
	MaxOpenConns int
	MaxIdleConns int
	MaxLifetime  time.Duration
}

// DefaultSQLXStoreConfig 默认配置
func DefaultSQLXStoreConfig() *SQLXStoreConfig {
	return &SQLXStoreConfig{
		TableName:    defaultSQLXTableName,
		KeyColumn:    defaultSQLXKeyColumn,
		DataColumn:   defaultSQLXDataColumn,
		VerColumn:    defaultSQLXVerColumn,
		DelColumn:    defaultSQLXDelColumn,
		TimeColumn:   defaultSQLXTimeColumn,
		MaxOpenConns: defaultSQLXMaxOpenConns,
		MaxIdleConns: defaultSQLXMaxIdleConns,
		MaxLifetime:  defaultSQLXMaxLifetime,
	}
}

// SQLXStore SQLX存储实现
type SQLXStore struct {
	db         *sqlx.DB
	config     *SQLXStoreConfig
	entityType Entity
}

// NewSQLXStore 创建SQLX存储
func NewSQLXStore(dsn string, entityType Entity, configs ...*SQLXStoreConfig) (*SQLXStore, error) {
	cfg := DefaultSQLXStoreConfig()
	if len(configs) > 0 && configs[0] != nil {
		cfg = configs[0]
	}
	cfg.DSN = dsn

	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("connect database failed: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.MaxLifetime)

	store := &SQLXStore{
		db:         db,
		config:     cfg,
		entityType: entityType,
	}

	if err := store.createTable(); err != nil {
		return nil, fmt.Errorf("create table failed: %w", err)
	}

	return store, nil
}

func (s *SQLXStore) createTable() error {
	formatSql := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		%s VARCHAR(%d) PRIMARY KEY,
		%s JSON NOT NULL,
		%s BIGINT DEFAULT 1,
		%s TINYINT DEFAULT 0,
		%s TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		INDEX idx_updated_at (%s),
		INDEX idx_is_deleted (%s)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		s.config.TableName,
		s.config.KeyColumn,
		defaultSQLXKeyVarcharLen,
		s.config.DataColumn,
		s.config.VerColumn,
		s.config.DelColumn,
		s.config.TimeColumn,
		s.config.TimeColumn,
		s.config.DelColumn,
	)

	_, err := s.db.Exec(formatSql)
	return err
}

func (s *SQLXStore) Get(ctx context.Context, key string) (Entity, error) {
	query := fmt.Sprintf(
		"SELECT %s, %s FROM %s WHERE %s = ? AND %s = 0",
		s.config.DataColumn,
		s.config.VerColumn,
		s.config.TableName,
		s.config.KeyColumn,
		s.config.DelColumn,
	)

	row := s.db.QueryRowxContext(ctx, query, key)
	dest := make(map[string]interface{})
	if err := row.MapScan(dest); err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	data, err := toBytes(dest[s.config.DataColumn])
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", s.config.DataColumn, err)
	}

	entity := s.entityType.Copy()
	if err := json.Unmarshal(data, entity); err != nil {
		return nil, err
	}

	return entity, nil
}

func (s *SQLXStore) MGet(ctx context.Context, keys []string) (map[string]Entity, error) {
	if len(keys) == 0 {
		return make(map[string]Entity), nil
	}

	query := fmt.Sprintf(
		"SELECT %s, %s, %s FROM %s WHERE %s IN (?) AND %s = 0",
		s.config.KeyColumn,
		s.config.DataColumn,
		s.config.VerColumn,
		s.config.TableName,
		s.config.KeyColumn,
		s.config.DelColumn,
	)

	query, args, err := sqlx.In(query, keys)
	if err != nil {
		return nil, err
	}

	query = s.db.Rebind(query)

	rows, err := s.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entities := make(map[string]Entity)
	for rows.Next() {
		dest := make(map[string]interface{})
		if err := rows.MapScan(dest); err != nil {
			return entities, fmt.Errorf("scan row: %w", err)
		}

		keyVal, err := toBytes(dest[s.config.KeyColumn])
		if err != nil {
			return entities, fmt.Errorf("read %s: %w", s.config.KeyColumn, err)
		}
		data, err := toBytes(dest[s.config.DataColumn])
		if err != nil {
			return entities, fmt.Errorf("read %s: %w", s.config.DataColumn, err)
		}

		entity := s.entityType.Copy()
		if err := json.Unmarshal(data, entity); err != nil {
			return entities, fmt.Errorf("unmarshal key %s: %w", string(keyVal), err)
		}
		entities[string(keyVal)] = entity
	}

	return entities, rows.Err()
}

func (s *SQLXStore) Insert(ctx context.Context, entity Entity) error {
	data, err := json.Marshal(entity)
	if err != nil {
		return err
	}

	// 使用 UPSERT 语义：若 key 已存在（如进程重启后 L1 清空导致 isNew 误判），
	// 执行 UPDATE 而非报主键冲突，避免永久重试循环。
	query := fmt.Sprintf(
		"INSERT INTO %s (%s, %s, %s, %s) VALUES (?, ?, ?, 0)"+
			" ON DUPLICATE KEY UPDATE %s = VALUES(%s), %s = VALUES(%s), %s = 0",
		s.config.TableName,
		s.config.KeyColumn, s.config.DataColumn, s.config.VerColumn, s.config.DelColumn,
		s.config.DataColumn, s.config.DataColumn,
		s.config.VerColumn, s.config.VerColumn,
		s.config.DelColumn,
	)

	_, err = s.db.ExecContext(ctx, query, entity.CacheKey(), data, entity.Version())
	return err
}

func (s *SQLXStore) Update(ctx context.Context, entity Entity) error {
	data, err := json.Marshal(entity)
	if err != nil {
		return err
	}

	// write-back 缓存场景：内存中可能多次写入后才刷盘，DB 版本可能落后多个版本号，
	// 用 < 而非 = 允许跨版本覆盖，同时仍能阻止低版本数据覆盖已落盘的高版本数据。
	query := fmt.Sprintf(
		"UPDATE %s SET %s = ?, %s = ?, %s = 0 WHERE %s = ? AND %s < ?",
		s.config.TableName,
		s.config.DataColumn,
		s.config.VerColumn,
		s.config.DelColumn,
		s.config.KeyColumn,
		s.config.VerColumn,
	)

	result, err := s.db.ExecContext(ctx, query, data, entity.Version(), entity.CacheKey(), entity.Version())
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("optimistic lock conflict: key=%s version=%d", entity.CacheKey(), entity.Version())
	}
	return nil
}

func (s *SQLXStore) Delete(ctx context.Context, key string) error {
	query := fmt.Sprintf(
		"UPDATE %s SET %s = 1 WHERE %s = ?",
		s.config.TableName,
		s.config.DelColumn,
		s.config.KeyColumn,
	)

	_, err := s.db.ExecContext(ctx, query, key)
	return err
}

func (s *SQLXStore) BatchInsert(ctx context.Context, entities []Entity) error {
	if len(entities) == 0 {
		return nil
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 与 Insert 保持一致：使用 UPSERT 语义防止 isNew 误判时的主键冲突循环
	query := fmt.Sprintf(
		"INSERT INTO %s (%s, %s, %s, %s) VALUES (?, ?, ?, 0)"+
			" ON DUPLICATE KEY UPDATE %s = VALUES(%s), %s = VALUES(%s), %s = 0",
		s.config.TableName,
		s.config.KeyColumn, s.config.DataColumn, s.config.VerColumn, s.config.DelColumn,
		s.config.DataColumn, s.config.DataColumn,
		s.config.VerColumn, s.config.VerColumn,
		s.config.DelColumn,
	)

	stmt, err := tx.Preparex(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, entity := range entities {
		data, err := json.Marshal(entity)
		if err != nil {
			return err
		}
		if _, err := stmt.ExecContext(ctx, entity.CacheKey(), data, entity.Version()); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLXStore) BatchUpdate(ctx context.Context, entities []Entity) error {
	if len(entities) == 0 {
		return nil
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 与 Update 保持一致：允许跨版本覆盖，阻止低版本覆盖高版本
	query := fmt.Sprintf(
		"UPDATE %s SET %s = ?, %s = ?, %s = 0 WHERE %s = ? AND %s < ?",
		s.config.TableName,
		s.config.DataColumn,
		s.config.VerColumn,
		s.config.DelColumn,
		s.config.KeyColumn,
		s.config.VerColumn,
	)

	stmt, err := tx.Preparex(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, entity := range entities {
		data, err := json.Marshal(entity)
		if err != nil {
			return err
		}
		result, err := stmt.ExecContext(ctx, data, entity.Version(), entity.CacheKey(), entity.Version())
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return fmt.Errorf("optimistic lock conflict: key=%s version=%d", entity.CacheKey(), entity.Version())
		}
	}

	return tx.Commit()
}

func (s *SQLXStore) BatchDelete(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	query := fmt.Sprintf(
		"UPDATE %s SET %s = 1 WHERE %s IN (?)",
		s.config.TableName,
		s.config.DelColumn,
		s.config.KeyColumn,
	)

	query, args, err := sqlx.In(query, keys)
	if err != nil {
		return err
	}

	query = s.db.Rebind(query)
	_, err = s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *SQLXStore) Close() error {
	return s.db.Close()
}

// toBytes 将 sqlx MapScan 返回的列值转换为 []byte
func toBytes(v interface{}) ([]byte, error) {
	switch val := v.(type) {
	case []byte:
		return val, nil
	case string:
		return []byte(val), nil
	case nil:
		return nil, fmt.Errorf("column value is nil")
	default:
		return nil, fmt.Errorf("unexpected column type %T", v)
	}
}

// CleanExpired 清理过期（软删除）数据
func (s *SQLXStore) CleanExpired(ctx context.Context, before time.Time) error {
	query := fmt.Sprintf(
		"DELETE FROM %s WHERE %s = 1 AND %s < ?",
		s.config.TableName,
		s.config.DelColumn,
		s.config.TimeColumn,
	)
	_, err := s.db.ExecContext(ctx, query, before)
	return err
}

// BeginTx 开启一个原生数据库事务，返回 DBStoreTx。
// SQLXStore 实现 TxCapableDBStore 接口。
func (s *SQLXStore) BeginTx(ctx context.Context) (DBStoreTx, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &SQLXStoreTx{tx: tx, config: s.config, entityType: s.entityType}, nil
}

// SQLXStoreTx 封装 sqlx.Tx，实现 DBStoreTx 接口。
type SQLXStoreTx struct {
	tx         *sqlx.Tx
	config     *SQLXStoreConfig
	entityType Entity
}

func (s *SQLXStoreTx) Get(ctx context.Context, key string) (Entity, error) {
	query := fmt.Sprintf(
		"SELECT %s, %s FROM %s WHERE %s = ? AND %s = 0",
		s.config.DataColumn,
		s.config.VerColumn,
		s.config.TableName,
		s.config.KeyColumn,
		s.config.DelColumn,
	)

	row := s.tx.QueryRowxContext(ctx, query, key)
	dest := make(map[string]interface{})
	if err := row.MapScan(dest); err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	data, err := toBytes(dest[s.config.DataColumn])
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", s.config.DataColumn, err)
	}

	entity := s.entityType.Copy()
	if err := json.Unmarshal(data, entity); err != nil {
		return nil, err
	}
	return entity, nil
}

func (s *SQLXStoreTx) Insert(ctx context.Context, entity Entity) error {
	data, err := json.Marshal(entity)
	if err != nil {
		return err
	}
	query := fmt.Sprintf(
		"INSERT INTO %s (%s, %s, %s, %s) VALUES (?, ?, ?, 0)"+
			" ON DUPLICATE KEY UPDATE %s = VALUES(%s), %s = VALUES(%s), %s = 0",
		s.config.TableName,
		s.config.KeyColumn, s.config.DataColumn, s.config.VerColumn, s.config.DelColumn,
		s.config.DataColumn, s.config.DataColumn,
		s.config.VerColumn, s.config.VerColumn,
		s.config.DelColumn,
	)
	_, err = s.tx.ExecContext(ctx, query, entity.CacheKey(), data, entity.Version())
	return err
}

func (s *SQLXStoreTx) Update(ctx context.Context, entity Entity) error {
	data, err := json.Marshal(entity)
	if err != nil {
		return err
	}
	query := fmt.Sprintf(
		"UPDATE %s SET %s = ?, %s = ?, %s = 0 WHERE %s = ? AND %s < ?",
		s.config.TableName,
		s.config.DataColumn,
		s.config.VerColumn,
		s.config.DelColumn,
		s.config.KeyColumn,
		s.config.VerColumn,
	)
	result, err := s.tx.ExecContext(ctx, query, data, entity.Version(), entity.CacheKey(), entity.Version())
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("optimistic lock conflict: key=%s version=%d", entity.CacheKey(), entity.Version())
	}
	return nil
}

func (s *SQLXStoreTx) Delete(ctx context.Context, key string) error {
	query := fmt.Sprintf(
		"UPDATE %s SET %s = 1 WHERE %s = ?",
		s.config.TableName,
		s.config.DelColumn,
		s.config.KeyColumn,
	)
	_, err := s.tx.ExecContext(ctx, query, key)
	return err
}

func (s *SQLXStoreTx) Commit() error {
	return s.tx.Commit()
}

func (s *SQLXStoreTx) Rollback() error {
	return s.tx.Rollback()
}

// 编译期检查：SQLXStore 实现 TxCapableDBStore
var _ TxCapableDBStore = (*SQLXStore)(nil)
