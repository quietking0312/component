package mds

import (
	"fmt"
	"math/rand"
	"runtime"
	"testing"
)

// 生成 n 条随机中文敏感词，每条 1~maxLen 个汉字
func genWords(n, maxLen int) []string {
	const base = 0x4E00 // CJK 统一汉字起始码点
	const size = 0x9FA5 - 0x4E00 + 1
	r := rand.New(rand.NewSource(42))
	words := make([]string, n)
	for i := range words {
		l := r.Intn(maxLen) + 1
		runes := make([]rune, l)
		for j := range runes {
			runes[j] = rune(base + r.Intn(size))
		}
		words[i] = string(runes)
	}
	return words
}

// 生成一段随机中文文本
func genText(length int) string {
	const base = 0x4E00
	const size = 0x9FA5 - 0x4E00 + 1
	r := rand.New(rand.NewSource(99))
	runes := make([]rune, length)
	for i := range runes {
		runes[i] = rune(base + r.Intn(size))
	}
	return string(runes)
}

// 模拟真实敏感词：从少量词根出发，通过追加后缀扩展，产生大量共享前缀的词
func genRealLikeWords(n, maxLen int) []string {
	const base = 0x4E00
	const size = 0x9FA5 - 0x4E00 + 1
	r := rand.New(rand.NewSource(42))
	roots := 200 // 200 个词根，每个词根派生多个词
	rootWords := make([][]rune, roots)
	for i := range rootWords {
		l := r.Intn(3) + 1
		runes := make([]rune, l)
		for j := range runes {
			runes[j] = rune(base + r.Intn(size))
		}
		rootWords[i] = runes
	}
	words := make([]string, 0, n)
	for len(words) < n {
		root := rootWords[r.Intn(roots)]
		extra := r.Intn(maxLen-len(root)) + 0
		w := make([]rune, len(root)+extra)
		copy(w, root)
		for i := len(root); i < len(w); i++ {
			w[i] = rune(base + r.Intn(size))
		}
		words = append(words, string(w))
	}
	return words[:n]
}

var (
	benchWords     = genWords(10000, 10)
	benchRealWords = genRealLikeWords(10000, 10)
	benchText      = genText(500)
)

// ---------- 内存占用 ----------

// allocBytes 统计累计分配量（含已释放的临时对象，反映构建开销）
func allocBytes(f func()) uint64 {
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	f()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

// liveBytes 统计结构体构建完成、GC 回收临时对象后的驻留堆内存（HeapAlloc）
// f 需返回要持有的对象，防止被提前回收
func liveBytes(f func() any) uint64 {
	runtime.GC()
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	obj := f()
	runtime.GC()
	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	runtime.KeepAlive(obj) // 必须在 ReadMemStats 之后，确保 obj 在测量期间存活
	if after.HeapAlloc > before.HeapAlloc {
		return after.HeapAlloc - before.HeapAlloc
	}
	return 0
}

type memResult struct {
	alloc uint64 // 累计分配（含临时对象）
	live  uint64 // GC 后驻留堆内存
}

func memStats(words []string) (mt, tr memResult) {
	mt.alloc = allocBytes(func() {
		m := NewMapTree()
		for _, w := range words {
			m.AddWord(w)
		}
		_ = m
	})
	mt.live = liveBytes(func() any {
		m := NewMapTree()
		for _, w := range words {
			m.AddWord(w)
		}
		return m
	})

	tr.alloc = allocBytes(func() {
		t := NewTrie()
		for _, w := range words {
			t.Insert(w)
		}
		t.Build()
		_ = t
	})
	tr.live = liveBytes(func() any {
		t := NewTrie()
		for _, w := range words {
			t.Insert(w)
		}
		t.Build()
		return t
	})
	return
}

func printMemResult(t *testing.T, name string, r memResult) {
	t.Logf("%-8s  alloc(含临时): %s  live(GC后驻留): %s", name, formatBytes(r.alloc), formatBytes(r.live))
}

func TestMemUsage(t *testing.T) {
	mt, tr := memStats(benchWords)
	t.Logf("=== 随机词（前缀几乎不共享）===")
	printMemResult(t, "MapTree", mt)
	printMemResult(t, "Trie", tr)

	mt2, tr2 := memStats(benchRealWords)
	t.Logf("=== 真实词（大量共享前缀）===")
	printMemResult(t, "MapTree", mt2)
	printMemResult(t, "Trie", tr2)
}

func formatBytes(b uint64) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%.2f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.2f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// ---------- 搜索速度 ----------

func BenchmarkMapTree_Search(b *testing.B) {
	mt := NewMapTree()
	for _, w := range benchWords {
		mt.AddWord(w)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mt.Load(benchText)
	}
}

func BenchmarkTrie_Match(b *testing.B) {
	tr := NewTrie()
	for _, w := range benchWords {
		tr.Insert(w)
	}
	tr.Build()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Match(benchText)
	}
}
