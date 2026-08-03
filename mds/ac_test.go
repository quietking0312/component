package mds

import (
	"reflect"
	"testing"
)

func buildAC(words []string) *AC {
	ac := NewAC()
	for _, w := range words {
		ac.AddWord(w)
	}
	ac.Build()
	return ac
}

func TestAC_Basic(t *testing.T) {
	ac := buildAC([]string{"he", "she", "his", "hers"})
	got := ac.Search("ushers")
	// 期望匹配：she[1,3], he[2,3], hers[2,5]
	want := [][2]int{{1, 3}, {2, 3}, {2, 5}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAC_Chinese(t *testing.T) {
	ac := buildAC([]string{"北京", "北京大学", "大学"})
	got := ac.Search("北京大学真不错")
	// 期望：北京[0,1], 北京大学[0,3], 大学[2,3]
	want := [][2]int{{0, 1}, {0, 3}, {2, 3}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAC_NoMatch(t *testing.T) {
	ac := buildAC([]string{"abc", "xyz"})
	got := ac.Search("hello world")
	if len(got) != 0 {
		t.Errorf("expected no match, got %v", got)
	}
}

func TestAC_Overlap(t *testing.T) {
	ac := buildAC([]string{"a", "aa", "aaa"})
	got := ac.Search("aaa")
	// 匹配：a[0,0], a[1,1], aa[0,1], a[2,2], aa[1,2], aaa[0,2]
	want := [][2]int{{0, 0}, {0, 1}, {1, 1}, {0, 2}, {1, 2}, {2, 2}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAC_EmptyText(t *testing.T) {
	ac := buildAC([]string{"abc"})
	got := ac.Search("")
	if len(got) != 0 {
		t.Errorf("expected no match on empty text, got %v", got)
	}
}
