package mcachedb

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

var (
	_userCache *MultiCache
	_bagCache  *MultiCache
	_taskCache *MultiCache
)

type UserEntity struct {
	*BaseEntity
	Uid int
	Gid int
}

func (u *UserEntity) Copy() Entity {
	return &UserEntity{
		BaseEntity: u.BaseEntity.Copy(),
		Uid:        u.Uid,
		Gid:        u.Gid,
	}
}

func (u *UserEntity) Marshal() ([]byte, error) {
	return json.Marshal(u)
}

func (u *UserEntity) Unmarshal(data []byte) error {
	return json.Unmarshal(data, u)
}

type BagData struct {
	Uid    int
	ItemId int
	Count  int
}

type TaskData struct {
	Uid       int
	ProcessId int
	Reward    int
}

func InitUserData() {
	cfg := &MultiCacheConfig{
		L1MaxSize:     1000,
		SyncInterval:  100 * time.Millisecond,
		FlushInterval: 100 * time.Millisecond,
	}
	_userCache, _ = NewMultiCache(NewMockDBStore(), nil, cfg)
	_bagCache, _ = NewMultiCache(NewMockDBStore(), nil, cfg)
	_taskCache, _ = NewMultiCache(NewMockDBStore(), nil, cfg)
	uid := 10001
	if err := _userCache.Set(&UserEntity{
		BaseEntity: NewBaseEntity(fmt.Sprintf("user:%d", uid)),
		Uid:        uid,
		Gid:        200,
	}); err != nil {
		fmt.Printf("init user:%d err:%v\n", uid, err)
	}
	var bagEntity = NewGenericEntity(fmt.Sprintf("bag:%d:%d", uid, 1000), &BagData{
		Uid:    uid,
		ItemId: 1000,
		Count:  1,
	})
	_bagCache.Set(bagEntity)
	var TaskEntity = NewEntityWrapper(fmt.Sprintf("task:%d:%d", uid, 100), &TaskData{
		Uid:       uid,
		ProcessId: 0,
		Reward:    0,
	})
	_taskCache.Set(TaskEntity)
}

func TestNewDistTx(t *testing.T) {
	InitUserData()
	uid := 10001

	tx := NewDistTx()
	userEnv, _ := _userCache.Get(fmt.Sprintf("user:%d", uid))
	user := userEnv.(*UserEntity)
	user.Gid += 10
	tx.AddSet(_userCache, user)
	bagEnv, _ := _bagCache.Get(fmt.Sprintf("bag:%d:%d", uid, 1000))
	bag := bagEnv.(*GenericEntity[*BagData])
	bag.Payload.Count -= 5
	tx.AddSet(_bagCache, bag)
	taskEnv, _ := _taskCache.Get(fmt.Sprintf("task:%d:%d", uid, 100))
	task := taskEnv.(*EntityWrapper)
	task.Data.(*TaskData).ProcessId += 5
	tx.AddSet(_taskCache, task)
	if err := tx.Commit(); err != nil {
		t.Error(err)
	}

	fmt.Println(_userCache.Get(fmt.Sprintf("user:%d", uid)))
	newBag, _ := _bagCache.Get(fmt.Sprintf("bag:%d:%d", uid, 1000))
	fmt.Println(newBag.(*GenericEntity[*BagData]).Payload)
	newTask, _ := _taskCache.Get(fmt.Sprintf("task:%d:%d", uid, 100))
	fmt.Println(newTask.(*EntityWrapper).Data)

}
