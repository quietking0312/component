package mmongo

import (
	"sort"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// defaultIndexTagName 是 Collection[T] 未通过 SetIndexTagName 配置时，
// EnsureIndexes 读取的结构体 tag 名。它是一个常量而不是包级变量，
// 这样并发的 Collection[T] 实例之间永远不会因为"当前生效的 tag 名是什么"
// 产生数据竞争。
//
// Tag 语法（结构体字段上以逗号分隔的多个 token）：
//
//	mindex:"index"            单字段升序索引
//	mindex:"unique"           单字段唯一索引（隐含 index）
//	mindex:"desc"             使用降序而不是升序
//	mindex:"sparse"           标记为稀疏索引
//	mindex:"text"             该字段参与集合的全文索引
//	mindex:"expire=3600"      TTL 索引，expireAfterSeconds=3600（字段必须是日期类型）
//	mindex:"group=name,seq=1" 该字段属于复合索引 "name"，按 seq 排序
//
// token 可以组合使用，例如 `mindex:"group=tenant_email,seq=2,unique,desc"`。
const defaultIndexTagName = "mindex"

type fieldIndexSpec struct {
	key    string
	desc   bool
	unique bool
	sparse bool
	text   bool
	expire *int32
	group  string
	seq    int
}

func parseFieldIndexSpec(bsonTag, indexTag string, declOrder int) (fieldIndexSpec, bool) {
	key := bsonKeyName(bsonTag)
	if key == "" || indexTag == "" {
		return fieldIndexSpec{}, false
	}

	spec := fieldIndexSpec{key: key, seq: declOrder}
	for _, tok := range strings.Split(indexTag, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		name, value, hasValue := strings.Cut(tok, "=")
		switch name {
		case "index":
			// 无需处理：tag 存在本身就已经标记了该字段要建索引
		case "unique":
			spec.unique = true
		case "desc":
			spec.desc = true
		case "sparse":
			spec.sparse = true
		case "text":
			spec.text = true
		case "group":
			if hasValue {
				spec.group = value
			}
		case "seq":
			if hasValue {
				if n, err := strconv.Atoi(value); err == nil {
					spec.seq = n
				}
			}
		case "expire":
			if hasValue {
				if n, err := strconv.Atoi(value); err == nil {
					seconds := int32(n)
					spec.expire = &seconds
				}
			}
		}
	}
	return spec, true
}

// bsonKeyName 从 `bson:"..."` tag 中解析出实际生效的 BSON 字段名；
// 当字段被排除（"-"）时返回空字符串。
func bsonKeyName(bsonTag string) string {
	name, _, _ := strings.Cut(bsonTag, ",")
	if name == "-" {
		return ""
	}
	return name
}

// indexModelsFor 根据 T 上以 tagName 声明的结构体 tag 构建
// *mongo.IndexModel 切片。T 必须是结构体类型（或指向结构体的指针）。
func indexModelsFor[T any](tagName string) ([]mongo.IndexModel, error) {
	typ, err := structTypeOf[T]()
	if err != nil {
		return nil, err
	}

	var (
		single    []fieldIndexSpec
		textKeys  []string
		groups    = map[string][]fieldIndexSpec{}
		groupSeen []string
	)

	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}
		indexTag, ok := f.Tag.Lookup(tagName)
		if !ok {
			continue
		}
		spec, ok := parseFieldIndexSpec(f.Tag.Get("bson"), indexTag, i)
		if !ok {
			continue
		}

		if spec.text {
			textKeys = append(textKeys, spec.key)
			continue
		}
		if spec.group != "" {
			if _, seen := groups[spec.group]; !seen {
				groupSeen = append(groupSeen, spec.group)
			}
			groups[spec.group] = append(groups[spec.group], spec)
			continue
		}
		single = append(single, spec)
	}

	var models []mongo.IndexModel

	for _, spec := range single {
		keys := bson.D{{Key: spec.key, Value: direction(spec.desc)}}
		opts := options.Index()
		if spec.unique {
			opts.SetUnique(true)
		}
		if spec.sparse {
			opts.SetSparse(true)
		}
		if spec.expire != nil {
			opts.SetExpireAfterSeconds(*spec.expire)
		}
		models = append(models, mongo.IndexModel{Keys: keys, Options: opts})
	}

	for _, name := range groupSeen {
		fields := groups[name]
		sort.SliceStable(fields, func(i, j int) bool { return fields[i].seq < fields[j].seq })

		keys := bson.D{}
		opts := options.Index()
		for _, f := range fields {
			keys = append(keys, bson.E{Key: f.key, Value: direction(f.desc)})
			if f.unique {
				opts.SetUnique(true)
			}
			if f.sparse {
				opts.SetSparse(true)
			}
		}
		models = append(models, mongo.IndexModel{Keys: keys, Options: opts})
	}

	if len(textKeys) > 0 {
		keys := bson.D{}
		for _, key := range textKeys {
			keys = append(keys, bson.E{Key: key, Value: "text"})
		}
		models = append(models, mongo.IndexModel{Keys: keys})
	}

	return models, nil
}

func direction(desc bool) int32 {
	if desc {
		return -1
	}
	return 1
}
