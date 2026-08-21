package mmongo

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// testDatabase 返回一个 *mongo.Database，背后的 client 从未真正拨号连接过网络：
// Connect/Database/Collection 只是在本地构造 handle，所以不需要真实的 MongoDB
// 实例即可安全使用。
func testDatabase(t *testing.T) *mongo.Database {
	client, err := mongo.Connect(options.Client().SetHosts([]string{"127.0.0.1:27017"}))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	return client.Database("registry_test")
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()
	db := testDatabase(t)

	registered := RegisterCollection[plainModel](r, db)

	got, ok := CollectionFrom[plainModel](r)
	if !ok {
		t.Fatal("expected plainModel to be registered")
	}
	if got != registered {
		t.Fatalf("expected CollectionFrom to return the same *Collection instance")
	}
	if got.Name() != "plainmodel" {
		t.Fatalf("unexpected collection name: %s", got.Name())
	}
}

func TestRegistry_CollectionFrom_NotRegistered(t *testing.T) {
	r := NewRegistry()
	if _, ok := CollectionFrom[plainModel](r); ok {
		t.Fatal("expected CollectionFrom to report not-found for an unregistered type")
	}
}

func TestRegistry_MustCollectionFrom_Panics(t *testing.T) {
	r := NewRegistry()
	defer func() {
		if recover() == nil {
			t.Fatal("expected MustCollectionFrom to panic for an unregistered type")
		}
	}()
	MustCollectionFrom[plainModel](r)
}

func TestRegistry_LookupNeedsMatchingCollectionNameOption(t *testing.T) {
	r := NewRegistry()
	db := testDatabase(t)

	RegisterCollection[plainModel](r, db, SetCollectionName("value_models"))
	RegisterCollection[*plainModel](r, db, SetCollectionName("pointer_models"))

	// 被覆盖过的名字在查找时必须重复传入，否则 CollectionFrom 无法知道
	// 某次注册用了哪个覆盖名。
	valueColl, ok := CollectionFrom[plainModel](r, SetCollectionName("value_models"))
	if !ok || valueColl.Name() != "value_models" {
		t.Fatalf("expected value_models collection, got ok=%v name=%v", ok, valueColl)
	}

	ptrColl, ok := CollectionFrom[*plainModel](r, SetCollectionName("pointer_models"))
	if !ok || ptrColl.Name() != "pointer_models" {
		t.Fatalf("expected pointer_models collection, got ok=%v name=%v", ok, ptrColl)
	}

	// 不重复覆盖名时，查找会退回到默认解析出的名字（两者都是
	// "plainmodel"），而这个名字在这里从未被注册过。
	if _, ok := CollectionFrom[plainModel](r); ok {
		t.Fatal("expected lookup without the SetCollectionName override to miss")
	}
}

func TestRegistry_SameDefaultNameCollidesAcrossValueAndPointer(t *testing.T) {
	r := NewRegistry()
	db := testDatabase(t)

	// plainModel 和 *plainModel 都解析为默认名字 "plainmodel"，所以按名字
	// （而不是按 Go 类型）做 key 会让第二次注册覆盖第一次在该 key 下的值——
	// 它们本来就代表同一个物理 MongoDB 集合。用输了这次覆盖的类型参数
	// 再去查找，即使 map 里条目还存在，CollectionFrom 内部的类型断言也会失败。
	RegisterCollection[plainModel](r, db)
	RegisterCollection[*plainModel](r, db)

	if _, ok := CollectionFrom[plainModel](r); ok {
		t.Fatal("expected the plainModel entry to have been overwritten by *plainModel's registration")
	}
	if _, ok := CollectionFrom[*plainModel](r); !ok {
		t.Fatal("expected the *plainModel entry (the last one registered under \"plainmodel\") to be found")
	}
}
