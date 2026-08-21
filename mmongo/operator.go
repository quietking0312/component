package mmongo

// Update 操作符，用于 UpdateOne/UpdateMany/FindOneAndUpdate 等更新语句
const (
	OpSet         = "$set"         // 设置字段值，如 bson.M{OpSet: bson.M{"age": 18}}
	OpSetOnInsert = "$setOnInsert" // upsert 插入时才设置的字段，如 bson.M{OpSetOnInsert: bson.M{"createdAt": time.Now()}}
	OpUnset       = "$unset"       // 删除字段，如 bson.M{OpUnset: bson.M{"tempField": ""}}
	OpRename      = "$rename"      // 重命名字段，如 bson.M{OpRename: bson.M{"oldName": "newName"}}
	OpInc         = "$inc"         // 数值递增/递减，如 bson.M{OpInc: bson.M{"views": 1}}
	OpMul         = "$mul"         // 数值相乘，如 bson.M{OpMul: bson.M{"price": 1.1}}
	OpMin         = "$min"         // 仅当新值更小时才更新，如 bson.M{OpMin: bson.M{"lowScore": 60}}
	OpMax         = "$max"         // 仅当新值更大时才更新，如 bson.M{OpMax: bson.M{"highScore": 100}}
	OpCurrentDate = "$currentDate" // 设置为当前日期/时间戳，如 bson.M{OpCurrentDate: bson.M{"updatedAt": true}}
	OpPush        = "$push"        // 向数组追加元素，如 bson.M{OpPush: bson.M{"tags": "new"}}
	OpPop         = "$pop"         // 移除数组首/尾元素，如 bson.M{OpPop: bson.M{"tags": 1}}（1=尾部，-1=首部）
	OpPull        = "$pull"        // 按条件移除数组元素，如 bson.M{OpPull: bson.M{"tags": "old"}}
	OpPullAll     = "$pullAll"     // 移除数组中匹配指定值的所有元素，如 bson.M{OpPullAll: bson.M{"tags": bson.A{"a", "b"}}}
	OpAddToSet    = "$addToSet"    // 向数组追加不重复元素，如 bson.M{OpAddToSet: bson.M{"tags": "unique"}}
	OpEach        = "$each"        // 配合 $push/$addToSet 批量追加，如 bson.M{OpPush: bson.M{"tags": bson.M{OpEach: bson.A{"a", "b"}}}}
	OpPosition    = "$position"    // 配合 $push 指定插入位置，如 bson.M{OpPush: bson.M{"tags": bson.M{OpEach: bson.A{"a"}, OpPosition: 0}}}
	OpSlice       = "$slice"       // 配合 $push 限制数组长度，如 bson.M{OpPush: bson.M{"tags": bson.M{OpEach: bson.A{"a"}, OpSlice: -5}}}
	OpBit         = "$bit"         // 按位操作，如 bson.M{OpBit: bson.M{"flags": bson.M{"and": 5}}}
)

// Query/比较操作符，用于筛选条件
const (
	OpEq        = "$eq"        // 等于，如 bson.M{"status": bson.M{OpEq: "active"}}
	OpNe        = "$ne"        // 不等于，如 bson.M{"status": bson.M{OpNe: "deleted"}}
	OpGt        = "$gt"        // 大于，如 bson.M{"age": bson.M{OpGt: 18}}
	OpGte       = "$gte"       // 大于等于，如 bson.M{"age": bson.M{OpGte: 18}}
	OpLt        = "$lt"        // 小于，如 bson.M{"age": bson.M{OpLt: 60}}
	OpLte       = "$lte"       // 小于等于，如 bson.M{"age": bson.M{OpLte: 60}}
	OpIn        = "$in"        // 属于集合，如 bson.M{"status": bson.M{OpIn: bson.A{"active", "pending"}}}
	OpNin       = "$nin"       // 不属于集合，如 bson.M{"status": bson.M{OpNin: bson.A{"deleted", "banned"}}}
	OpExists    = "$exists"    // 字段是否存在，如 bson.M{"email": bson.M{OpExists: true}}
	OpType      = "$type"      // 字段 BSON 类型匹配，如 bson.M{"age": bson.M{OpType: "int"}}
	OpRegex     = "$regex"     // 正则匹配，如 bson.M{"name": bson.M{OpRegex: "^Al", "$options": "i"}}
	OpSize      = "$size"      // 数组长度匹配，如 bson.M{"tags": bson.M{OpSize: 3}}
	OpAll       = "$all"       // 数组包含全部指定元素，如 bson.M{"tags": bson.M{OpAll: bson.A{"a", "b"}}}
	OpElemMatch = "$elemMatch" // 数组元素匹配子条件，如 bson.M{"scores": bson.M{OpElemMatch: bson.M{OpGte: 80, OpLt: 90}}}
	OpMod       = "$mod"       // 取模匹配，如 bson.M{"qty": bson.M{OpMod: bson.A{4, 0}}}（qty % 4 == 0）
	OpText      = "$text"      // 全文检索，如 bson.M{OpText: bson.M{"$search": "mongodb"}}
	OpWhere     = "$where"     // JS 表达式匹配（不推荐，性能差），如 bson.M{OpWhere: "this.age > 18"}
)

// 逻辑操作符，用于组合查询条件
const (
	OpAnd = "$and" // 逻辑与，如 bson.M{OpAnd: bson.A{bson.M{"age": bson.M{OpGt: 18}}, bson.M{"status": "active"}}}
	OpOr  = "$or"  // 逻辑或，如 bson.M{OpOr: bson.A{bson.M{"status": "active"}, bson.M{"vip": true}}}
	OpNot = "$not" // 逻辑非，如 bson.M{"age": bson.M{OpNot: bson.M{OpGt: 18}}}
	OpNor = "$nor" // 逻辑或非，如 bson.M{OpNor: bson.A{bson.M{"status": "deleted"}, bson.M{"status": "banned"}}}
)

// 聚合管道阶段操作符，用于 Aggregate
const (
	OpMatch       = "$match"       // 过滤文档，如 bson.M{OpMatch: bson.M{"status": "active"}}
	OpGroup       = "$group"       // 分组统计，如 bson.M{OpGroup: bson.M{"_id": "$userId", "total": bson.M{"$sum": 1}}}
	OpProject     = "$project"     // 字段投影，如 bson.M{OpProject: bson.M{"name": 1, "age": 1, "_id": 0}}
	OpSort        = "$sort"        // 排序，如 bson.M{OpSort: bson.M{"createdAt": -1}}
	OpLimit       = "$limit"       // 限制返回数量，如 bson.M{OpLimit: 10}
	OpSkip        = "$skip"        // 跳过指定数量，如 bson.M{OpSkip: 20}
	OpUnwind      = "$unwind"      // 展开数组字段，如 bson.M{OpUnwind: "$tags"}
	OpLookup      = "$lookup"      // 关联查询（类似 join），如 bson.M{OpLookup: bson.M{"from": "orders", "localField": "_id", "foreignField": "userId", "as": "orders"}}
	OpGraphLookup = "$graphLookup" // 递归关联查询，如 bson.M{OpGraphLookup: bson.M{"from": "employees", "startWith": "$reportsTo", "connectFromField": "reportsTo", "connectToField": "_id", "as": "chain"}}
	OpAddFields   = "$addFields"   // 追加计算字段，如 bson.M{OpAddFields: bson.M{"fullName": bson.M{"$concat": bson.A{"$first", " ", "$last"}}}}
	OpReplaceRoot = "$replaceRoot" // 替换根文档，如 bson.M{OpReplaceRoot: bson.M{"newRoot": "$profile"}}
	OpFacet       = "$facet"       // 多维度并行聚合，如 bson.M{OpFacet: bson.M{"byStatus": bson.A{bson.M{OpGroup: bson.M{"_id": "$status", "count": bson.M{"$sum": 1}}}}}}
	OpBucket      = "$bucket"      // 按区间分桶，如 bson.M{OpBucket: bson.M{"groupBy": "$age", "boundaries": bson.A{0, 18, 60, 120}}}
	OpSortByCount = "$sortByCount" // 分组计数并排序，如 bson.M{OpSortByCount: "$status"}
	OpCount       = "$count"       // 统计文档数量，如 bson.M{OpCount: "total"}
	OpSample      = "$sample"      // 随机采样，如 bson.M{OpSample: bson.M{"size": 5}}
	OpOut         = "$out"         // 将结果写入集合，如 bson.M{OpOut: "archivedOrders"}
	OpMerge       = "$merge"       // 将结果合并写入集合，如 bson.M{OpMerge: bson.M{"into": "summary", "on": "_id"}}
)
