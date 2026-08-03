# mds — 数据结构库

`github.com/quietking0312/component/mds`

---

## 数据结构一览

| 类型 | 文件 | 核心操作 |
|------|------|---------|
| 跳表 `SkipList` | skip_list.go | 查/插/删 O(log n) |
| 字典树 `MapTree` | map_tree.go | 构建 O(m)，扫描 O(n) |
| 前缀树 `Trie` | trie.go | 查/插/删 O(k)，多模式搜索 O(n + z) |

> n: 文本长度，m: 所有模式总长度，k: 单词长度，z: 匹配数量

---

## 跳表 SkipList

有序链表的概率平衡变体，支持泛型 key/value，通过随机层数实现 O(log n) 的查、插、删。

**适用**：排行榜、有序任务队列、需要有序遍历又频繁增删的场景。  
**不适用**：无序数据或只需哈希查找（直接用 map）。

```go
sl := mds.NewSkipList[int, string](func(a, b int) int {
    if a < b { return mds.Less }
    if a > b { return mds.Greater }
    return mds.Equal
})
sl.Insert(10, "ten")
sl.Insert(3, "three")
v, ok := sl.Search(10) // "ten", true
sl.Remove(3)
```

---

## 字典树 MapTree

专为**贪心单次扫描**设计，找到第一个匹配词立即跳过，不报告重叠命中。  
支持 `SetExcept` 跳过干扰字符（空格、标点等）。

**适用**：敏感词过滤、词库固定、只需要"命中/替换"而不需要所有重叠位置。  
**不适用**：需要找出所有重叠匹配，或词库需要动态增删。

```go
mt := mds.NewMapTree()
mt.SetExcept(func(r rune) bool { return r == ' ' })
mt.AddWord("敏感词")
positions := mt.Load("含有敏 感词的文本") // [][2]int
```

---

## 前缀树 Trie

通用前缀树，支持增删查和前缀枚举。内置 AC 自动机：调用 `Build()` 后可用 `Match()` 做多模式搜索，找出文本中**所有模式的所有出现位置**（含重叠）。  
Insert/Delete 后标记 dirty，下次 `Match()` 自动重建失配链。  
支持 `SetExcept` 跳过干扰字符。

**适用**：词库需要动态增删、需要所有重叠命中位置、前缀补全/枚举。  
**不适用**：只需贪心单次扫描，用 `MapTree` 更轻量。

```go
tr := mds.NewTrie()
tr.SetExcept(func(r rune) bool { return r == ' ' || r == '-' })
tr.Insert("敏感词")
tr.Insert("违禁内容")

// 前缀树操作
tr.Search("敏感词")           // true
tr.StartsWith("敏感")         // true
tr.WordsWithPrefix("违禁")    // ["违禁内容"]

// 多模式搜索（自动 Build）
hits := tr.Match("含有敏 感词的文本") // [][2]int，跳过空格后命中

// 动态更新，下次 Match 自动重建
tr.Delete("敏感词")
tr.Insert("新词")
```

---

## 选型速查

```
需要有序存储 + 频繁增删？
└── SkipList

文本中搜索多个模式词？
├── 词库固定，只需命中/替换（贪心）→ MapTree
└── 需要所有重叠位置，或词库动态变化 → Trie
    ├── 还需要前缀查询/枚举 → Trie
    └── 词库纯静态 → Trie（Build 一次，反复 Match）
```

---

## 性能测试

测试环境：Windows 10，12 核 CPU，Go 1.25。

### 内存占用

使用 `runtime.MemStats` 测量：`alloc` 为构建过程累计分配（含临时对象），`live` 为 GC 回收临时对象后的堆驻留量。

**合成数据（10,000 条，每条最多 10 个汉字）**

| 场景 | 结构 | alloc（含临时） | live（GC 后驻留） |
|------|------|---------------|----------------|
| 随机词（前缀几乎不共享） | MapTree | 9.80 MB | 9.49 MB |
| 随机词 | Trie | 12.91 MB | 11.11 MB |
| 真实词（大量共享前缀） | MapTree | 6.34 MB | 6.12 MB |
| 真实词 | Trie | 8.43 MB | 7.22 MB |

**真实屏蔽词库（15,985 条）**

| 结构 | alloc（含临时） | live（GC 后驻留） |
|------|---------------|----------------|
| MapTree | 7.65 MB | 7.46 MB |
| Trie | 10.35 MB | 8.38 MB |

MapTree 节点只存 `Children + IsEnd`，Trie 节点额外携带 `fail / output / pattern` 字段，单节点更大，所以内存占用高约 15~20%。

### 搜索速度

**合成数据（随机 10,000 词，500 字文本，低命中率）**

| 结构 | ns/op | B/op | allocs/op |
|------|-------|------|-----------|
| MapTree | ~17,476 | 3,056 | 7 |
| Trie | ~16,976 | 3,058 | 7 |

两者持平，误差范围内无显著差异。

**真实屏蔽词库（15,985 词，高命中率文本）**

| 结构 | ns/op | B/op | allocs/op |
|------|-------|------|-----------|
| MapTree | ~17,243 | 10,224 | 10 |
| Trie | ~23,049 | 10,230 | 10 |

高命中率下 MapTree 比 Trie 快约 **34%**。原因是 MapTree 贪心匹配——每命中一个词就跳过整段，迭代次数少；Trie 需要报告所有重叠命中，命中率越高开销越大。两者语义不同，MapTree 不输出重叠位置。
