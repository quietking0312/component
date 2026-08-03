package mds

import (
	"reflect"
	"sort"
	"testing"
)

func TestTrie_SearchAndInsert(t *testing.T) {
	tr := NewTrie()
	tr.Insert("apple")
	tr.Insert("app")
	tr.Insert("application")

	if !tr.Search("apple") {
		t.Error("expected to find 'apple'")
	}
	if !tr.Search("app") {
		t.Error("expected to find 'app'")
	}
	if tr.Search("ap") {
		t.Error("'ap' should not exist")
	}
}

func TestTrie_StartsWith(t *testing.T) {
	tr := NewTrie()
	tr.Insert("apple")
	tr.Insert("application")

	if !tr.StartsWith("app") {
		t.Error("expected prefix 'app' to exist")
	}
	if tr.StartsWith("xyz") {
		t.Error("prefix 'xyz' should not exist")
	}
}

func TestTrie_Delete(t *testing.T) {
	tr := NewTrie()
	tr.Insert("apple")
	tr.Insert("app")

	if !tr.Delete("apple") {
		t.Error("expected delete to return true")
	}
	if tr.Search("apple") {
		t.Error("'apple' should be deleted")
	}
	// 删除 apple 后 app 仍然存在
	if !tr.Search("app") {
		t.Error("'app' should still exist after deleting 'apple'")
	}
	// 再删除 app
	tr.Delete("app")
	if tr.StartsWith("app") {
		t.Error("prefix 'app' should be gone after deleting all words")
	}
}

func TestTrie_DeleteNonExistent(t *testing.T) {
	tr := NewTrie()
	tr.Insert("hello")
	if tr.Delete("world") {
		t.Error("deleting non-existent word should return false")
	}
}

func TestTrie_WordsWithPrefix(t *testing.T) {
	tr := NewTrie()
	words := []string{"go", "golang", "gone", "good", "bad"}
	for _, w := range words {
		tr.Insert(w)
	}

	got := tr.WordsWithPrefix("go")
	sort.Strings(got)
	want := []string{"go", "golang", "gone", "good"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d]=%s, want %s", i, got[i], want[i])
		}
	}
}

func TestTrie_Chinese(t *testing.T) {
	tr := NewTrie()
	tr.Insert("北京")
	tr.Insert("北京大学")
	tr.Insert("北大")

	if !tr.Search("北京") {
		t.Error("expected to find '北京'")
	}
	if !tr.StartsWith("北") {
		t.Error("expected prefix '北' to exist")
	}

	got := tr.WordsWithPrefix("北京")
	sort.Strings(got)
	want := []string{"北京", "北京大学"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTrie_Match_Basic(t *testing.T) {
	tr := NewTrie()
	tr.Insert("he")
	tr.Insert("she")
	tr.Insert("his")
	tr.Insert("hers")
	// Build 由 Match 内部自动触发
	got := tr.Match("ushers")
	want := [][2]int{{1, 3}, {2, 3}, {2, 5}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestTrie_Match_Chinese(t *testing.T) {
	tr := NewTrie()
	tr.Insert("北京")
	tr.Insert("北京大学")
	tr.Insert("大学")
	got := tr.Match("北京大学真不错")
	want := [][2]int{{0, 1}, {0, 3}, {2, 3}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestTrie_Match_Except(t *testing.T) {
	tr := NewTrie()
	tr.Insert("北京大学")
	tr.Insert("大学")
	// 空格和标点不参与匹配
	tr.SetExcept(func(r rune) bool {
		return r == ' ' || r == '，' || r == '。'
	})

	// "北 京 大 学" 中间有空格，过滤后等同于"北京大学"
	got := tr.Match("北 京 大 学真不错")
	// 北京大学 匹配到原文 [0,6]（0=北, 6=学），大学匹配到原文 [4,6]
	want := [][2]int{{0, 6}, {4, 6}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestTrie_Match_ExceptNoFalsePositive(t *testing.T) {
	tr := NewTrie()
	tr.Insert("abcd")
	tr.SetExcept(func(r rune) bool { return r == '-' })

	// "ab-cd" 过滤 '-' 后等同于 "abcd"，应命中
	got := tr.Match("ab-cd")
	if len(got) != 1 || got[0] != [2]int{0, 4} {
		t.Errorf("got %v, want [[0 4]]", got)
	}

	// "ab--ef" 过滤后是 "abef"，不应命中
	got2 := tr.Match("ab--ef")
	if len(got2) != 0 {
		t.Errorf("expected no match, got %v", got2)
	}
}

func TestTrie_Match_AfterDelete(t *testing.T) {
	tr := NewTrie()
	tr.Insert("bad")
	tr.Insert("badly")
	tr.Build()

	tr.Delete("bad")
	// Delete 后 dirty=true，Match 会自动 rebuild
	got := tr.Match("that was badly done")
	if len(got) != 1 || got[0] != [2]int{9, 13} {
		t.Errorf("got %v, want [[9 13]]", got)
	}
}
