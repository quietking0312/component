package mmongo

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// resolveIndexOptions 把 v2 *options.IndexOptionsBuilder 里延迟执行的
// setter 应用到一个新的 IndexOptions 上，方便测试对结果做断言。
func resolveIndexOptions(t *testing.T, b *options.IndexOptionsBuilder) *options.IndexOptions {
	opts := &options.IndexOptions{}
	if b == nil {
		return opts
	}
	for _, fn := range b.List() {
		if err := fn(opts); err != nil {
			t.Fatalf("apply index option: %v", err)
		}
	}
	return opts
}

type indexTagUser struct {
	ID        string `bson:"_id,omitempty"`
	Email     string `bson:"email" mindex:"unique"`
	Nickname  string `bson:"nickname" mindex:"index,sparse"`
	Bio       string `bson:"bio" mindex:"text"`
	Summary   string `bson:"summary" mindex:"text"`
	Tenant    string `bson:"tenant" mindex:"group=tenant_email,seq=1"`
	GroupMail string `bson:"group_mail" mindex:"group=tenant_email,seq=2,unique,desc"`
	CreatedAt int64  `bson:"created_at" mindex:"expire=3600"`
	Ignored   string `bson:"ignored"`
}

func TestIndexModelsFor(t *testing.T) {
	models, err := indexModelsFor[indexTagUser](defaultIndexTagName)
	if err != nil {
		t.Fatal(err)
	}

	// 单字段：email(unique)、nickname(sparse)、created_at(ttl)
	// 复合：tenant_email
	// 全文：bio + summary 合并
	if len(models) != 5 {
		t.Fatalf("expected 5 index models, got %d", len(models))
	}

	var (
		foundEmail    bool
		foundNickname bool
		foundExpire   bool
		foundGroup    bool
		foundText     bool
	)

	for _, m := range models {
		keys, ok := m.Keys.(bson.D)
		if !ok || len(keys) == 0 {
			t.Fatalf("unexpected keys type/len: %#v", m.Keys)
		}

		opts := resolveIndexOptions(t, m.Options)

		switch keys[0].Key {
		case "email":
			foundEmail = true
			if opts.Unique == nil || !*opts.Unique {
				t.Errorf("email index expected unique=true, got %#v", opts)
			}
		case "nickname":
			foundNickname = true
			if opts.Sparse == nil || !*opts.Sparse {
				t.Errorf("nickname index expected sparse=true, got %#v", opts)
			}
		case "created_at":
			foundExpire = true
			if opts.ExpireAfterSeconds == nil || *opts.ExpireAfterSeconds != 3600 {
				t.Errorf("created_at index expected expireAfterSeconds=3600, got %#v", opts)
			}
		case "tenant":
			foundGroup = true
			if len(keys) != 2 || keys[1].Key != "group_mail" {
				t.Fatalf("unexpected compound index keys: %#v", keys)
			}
			if keys[1].Value != int32(-1) {
				t.Errorf("group_mail expected desc order, got %#v", keys[1].Value)
			}
			if opts.Unique == nil || !*opts.Unique {
				t.Errorf("tenant_email compound index expected unique=true, got %#v", opts)
			}
		case "bio":
			foundText = true
			if len(keys) != 2 || keys[1].Key != "summary" || keys[0].Value != "text" || keys[1].Value != "text" {
				t.Fatalf("unexpected text index keys: %#v", keys)
			}
		}
	}

	if !foundEmail || !foundNickname || !foundExpire || !foundGroup || !foundText {
		t.Fatalf("missing expected index(es): email=%v nickname=%v expire=%v group=%v text=%v",
			foundEmail, foundNickname, foundExpire, foundGroup, foundText)
	}
}

func TestIndexModelsFor_NoTags(t *testing.T) {
	type plain struct {
		Name string `bson:"name"`
	}
	models, err := indexModelsFor[plain](defaultIndexTagName)
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 0 {
		t.Fatalf("expected no index models, got %d", len(models))
	}
}

func TestIndexModelsFor_CustomTagName(t *testing.T) {
	type customTagModel struct {
		Email string `bson:"email" myindex:"unique"`
	}

	models, err := indexModelsFor[customTagModel]("myindex")
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 {
		t.Fatalf("expected 1 index model, got %d", len(models))
	}
	opts := resolveIndexOptions(t, models[0].Options)
	if opts.Unique == nil || !*opts.Unique {
		t.Fatalf("expected unique index, got %#v", opts)
	}
}
