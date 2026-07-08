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
	sql := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
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

	_, err := s.db.Exec(sql)
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

	var result struct {
		Data    []byte `db:"cache_data"`
		Version int64  `db:"version"`
	}

	err := s.db.GetContext(ctx, &result, query, key)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	entity := s.entityType.Copy()
	if err := json.Unmarshal(result.Data, entity); err != nil {
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

	var results []struct {
		Key     string `db:"cache_key"`
		Data    []byte `db:"cache_data"`
		Version int64  `db:"version"`
	}

	if err := s.db.SelectContext(ctx, &results, query, args...); err != nil {
		return nil, err
	}

	entities := make(map[string]Entity)
	for _, r := range results {
		entity := s.entityType.Copy()
		if err := json.Unmarshal(r.Data, entity); err != nil {
			continue
		}
		entities[r.Key] = entity
	}

	return entities, nil
}

func (s *SQLXStore) Insert(ctx context.Context, entity Entity) error {
	data, err := json.Marshal(entity)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s, %s, %s, %s) VALUES (?, ?, ?, 0)",
		s.config.TableName,
		s.config.KeyColumn,
		s.config.DataColumn,
		s.config.VerColumn,
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

	// WHERE version < ? 乐观锁：只有数据库里的版本比当前版本旧才允许覆盖，
	// 防止并发场景下低版本写操作静默覆盖已落盘的高版本数据。
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

	query := fmt.Sprintf(
		"INSERT INTO %s (%s, %s, %s, %s) VALUES (?, ?, ?, 0)",
		s.config.TableName,
		s.config.KeyColumn,
		s.config.DataColumn,
		s.config.VerColumn,
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

	// WHERE version < ? 乐观锁，与 Update 保持一致
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
