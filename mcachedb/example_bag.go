package mcachedb

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// ---------- BagItemEntity：每个道具作为独立 Entity ----------

type BagItemEntity struct {
	BaseEntity
	UID    int64 `json:"uid"    db:"uid"`
	ItemID int64 `json:"item_id" db:"item_id"`
	Count  int64 `json:"count"  db:"count"`
}

func NewBagItemEntity(uid, itemID, count int64) *BagItemEntity {
	return &BagItemEntity{
		BaseEntity: *NewBaseEntity(fmt.Sprintf("bag:%d:%d", uid, itemID)),
		UID:        uid,
		ItemID:     itemID,
		Count:      count,
	}
}

func (e *BagItemEntity) Copy() Entity {
	return &BagItemEntity{
		BaseEntity: *e.BaseEntity.Copy(),
		UID:        e.UID,
		ItemID:     e.ItemID,
		Count:      e.Count,
	}
}

func (e *BagItemEntity) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func (e *BagItemEntity) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

// ---------- BagDBStore：针对 user_bag 表的 DBStore ----------

type BagDBStore struct {
	db *sqlx.DB
}

func NewBagDBStore(db *sqlx.DB) *BagDBStore {
	return &BagDBStore{db: db}
}

func (s *BagDBStore) Get(ctx context.Context, key string) (Entity, error) {
	var uid, itemID int64
	if _, err := fmt.Sscanf(key, "bag:%d:%d", &uid, &itemID); err != nil {
		return nil, fmt.Errorf("invalid bag key: %s", key)
	}

	query := "SELECT uid, item_id, count FROM user_bag WHERE uid = ? AND item_id = ?"
	var item BagItemEntity
	if err := s.db.GetContext(ctx, &item, query, uid, itemID); err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, err
	}
	item.ID = key
	return &item, nil
}

func (s *BagDBStore) MGet(ctx context.Context, keys []string) (map[string]Entity, error) {
	result := make(map[string]Entity)
	if len(keys) == 0 {
		return result, nil
	}

	for _, key := range keys {
		ent, err := s.Get(ctx, key)
		if err != nil {
			continue
		}
		if ent != nil {
			result[key] = ent
		}
	}
	return result, nil
}

func (s *BagDBStore) Insert(ctx context.Context, entity Entity) error {
	item := entity.(*BagItemEntity)
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO user_bag (uid, item_id, count) VALUES (?, ?, ?)",
		item.UID, item.ItemID, item.Count,
	)
	return err
}

func (s *BagDBStore) Update(ctx context.Context, entity Entity) error {
	item := entity.(*BagItemEntity)
	_, err := s.db.ExecContext(ctx,
		"UPDATE user_bag SET count = ? WHERE uid = ? AND item_id = ?",
		item.Count, item.UID, item.ItemID,
	)
	return err
}

func (s *BagDBStore) Delete(ctx context.Context, key string) error {
	var uid, itemID int64
	if _, err := fmt.Sscanf(key, "bag:%d:%d", &uid, &itemID); err != nil {
		return fmt.Errorf("invalid bag key: %s", key)
	}
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM user_bag WHERE uid = ? AND item_id = ?",
		uid, itemID,
	)
	return err
}

func (s *BagDBStore) BatchInsert(ctx context.Context, entities []Entity) error {
	for _, e := range entities {
		if err := s.Insert(ctx, e); err != nil {
			return err
		}
	}
	return nil
}

func (s *BagDBStore) BatchUpdate(ctx context.Context, entities []Entity) error {
	for _, e := range entities {
		if err := s.Update(ctx, e); err != nil {
			return err
		}
	}
	return nil
}

func (s *BagDBStore) BatchDelete(ctx context.Context, keys []string) error {
	for _, k := range keys {
		if err := s.Delete(ctx, k); err != nil {
			return err
		}
	}
	return nil
}

func (s *BagDBStore) Close() error { return nil }

// ---------- BagService：业务层封装 ----------

type BagService struct {
	cache *MultiCache
}

func NewBagService(cache *MultiCache) *BagService {
	return &BagService{cache: cache}
}

// GetAll 使用 MGet 批量读取整个背包
// 注意：实际项目中 itemIDs 可以从索引表或 Redis Set 中获取
func (s *BagService) GetAll(uid int64, itemIDs []int64) (map[int64]*BagItemEntity, error) {
	keys := make([]string, len(itemIDs))
	for i, id := range itemIDs {
		keys[i] = fmt.Sprintf("bag:%d:%d", uid, id)
	}

	result, err := s.cache.MGet(keys)
	if err != nil {
		return nil, err
	}

	items := make(map[int64]*BagItemEntity, len(result))
	for _, ent := range result {
		item := ent.(*BagItemEntity)
		items[item.ItemID] = item
	}
	return items, nil
}

// ChangeCount 只修改单个道具，数据库仅更新一行
func (s *BagService) ChangeCount(uid, itemID, delta int64) error {
	key := fmt.Sprintf("bag:%d:%d", uid, itemID)
	ent, _ := s.cache.Get(key)

	var item *BagItemEntity
	if ent == nil {
		item = NewBagItemEntity(uid, itemID, 0)
	} else {
		item = ent.(*BagItemEntity)
	}

	item.Count += delta
	if item.Count <= 0 {
		return s.cache.Delete(key)
	}
	return s.cache.Set(item)
}

// SetItem 直接设置单个道具
func (s *BagService) SetItem(item *BagItemEntity) error {
	return s.cache.Set(item)
}

// DeleteItem 删除单个道具
func (s *BagService) DeleteItem(uid, itemID int64) error {
	key := fmt.Sprintf("bag:%d:%d", uid, itemID)
	return s.cache.Delete(key)
}

// ---------- 跨业务事务示例 ----------

// GoldEntity 金币实体（简化示例）
type GoldEntity struct {
	BaseEntity
	UID  int64 `json:"uid"`
	Gold int64 `json:"gold"`
}

func NewGoldEntity(uid, gold int64) *GoldEntity {
	return &GoldEntity{
		BaseEntity: *NewBaseEntity(fmt.Sprintf("gold:%d", uid)),
		UID:        uid,
		Gold:       gold,
	}
}
func (e *GoldEntity) Copy() Entity {
	return &GoldEntity{
		BaseEntity: *e.BaseEntity.Copy(),
		UID:        e.UID,
		Gold:       e.Gold,
	}
}

func (e *GoldEntity) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func (e *GoldEntity) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

// BuyItemExample 用 DistTx 实现扣金币 + 加背包道具
// goldCache 和 bagCache 是两个独立的 MultiCache
func BuyItemExample(goldCache, bagCache *MultiCache, uid int64, itemID int64, cost int64) error {
	// 1. 读取当前金币
	goldEnt, _ := goldCache.Get(fmt.Sprintf("gold:%d", uid))
	gold := NewGoldEntity(uid, 0)
	if goldEnt != nil {
		gold = goldEnt.(*GoldEntity)
	}
	if gold.Gold < cost {
		return fmt.Errorf("gold not enough")
	}
	gold.Gold -= cost

	// 2. 读取当前道具数量
	bagKey := fmt.Sprintf("bag:%d:%d", uid, itemID)
	bagEnt, _ := bagCache.Get(bagKey)
	bagItem := NewBagItemEntity(uid, itemID, 0)
	if bagEnt != nil {
		bagItem = bagEnt.(*BagItemEntity)
	}
	bagItem.Count += 1

	// 3. 跨缓存事务提交
	tx := NewDistTx()
	if err := tx.AddSet(goldCache, gold); err != nil {
		return err
	}
	if err := tx.AddSet(bagCache, bagItem); err != nil {
		return err
	}
	return tx.Commit()
}
