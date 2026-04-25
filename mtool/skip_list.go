package mtool

import (
	"fmt"
	"math/rand"
	"time"
)

/*
跳表

*/

const (
	Less    = -1
	Equal   = 0
	Greater = 1
)

type SkipListNode[K comparable, V any] struct {
	next  []*SkipListNode[K, V] // 每一层的下一跳指针
	key   K
	value V
}

func newNode[K comparable, V any](key K, val V, lv int) *SkipListNode[K, V] {
	return &SkipListNode[K, V]{next: make([]*SkipListNode[K, V], lv), key: key, value: val}
}

type SkipList[K comparable, V any] struct {
	head     *SkipListNode[K, V] // 头节点
	level    int                 // 当前最高层数
	maxLevel int                 // 最大层数
	p        float64             // 随机提升概率 1 百分百提升
	compare  func(a, b K) int
	rand     *rand.Rand
}

func NewSkipList[K comparable, V any](compare func(a, b K) int) *SkipList[K, V] {
	return &SkipList[K, V]{
		level:    1,
		maxLevel: 8,
		p:        0.5,
		head:     &SkipListNode[K, V]{next: make([]*SkipListNode[K, V], 8)},
		compare:  compare,
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (skipList *SkipList[K, V]) Insert(key K, value V) {
	update := make(map[int]*SkipListNode[K, V])
	curr := skipList.head

	for i := skipList.level - 1; i >= 0; i-- {

		for curr.next[i] != nil && skipList.compare(curr.next[i].key, key) == Less {
			curr = curr.next[i]
		}
		update[i] = curr

	}

	level := skipList.RandomLevel()
	if level > skipList.level {
		for i := skipList.level; i < level; i++ {
			update[i] = skipList.head
		}
		skipList.level = level
	}

	nNode := newNode(key, value, level)

	for i := 0; i < level; i++ {
		nNode.next[i] = update[i].next[i]
		update[i].next[i] = nNode
	}
}

func (skipList *SkipList[K, V]) RandomLevel() int {
	var level int = 1
	for skipList.rand.Float64() < skipList.p && level < skipList.maxLevel {
		level++
	}
	return level
}

func (skipList *SkipList[K, V]) Remove(key K) {

	update := make(map[int]*SkipListNode[K, V])
	curr := skipList.head
	for i := skipList.level - 1; i >= 0; i-- {
		for {
			if curr.next[i] == nil {
				break
			}
			if curr.next[i].key == key {
				update[i] = curr
				break
			}
			if skipList.compare(curr.next[i].key, key) == Less {
				curr = curr.next[i]
				continue
			} else {
				break
			}
		}
	}

	for i, v := range update {
		v.next[i] = v.next[i].next[i]
	}
	// 调整 level：如果最高层为空，则降低 level
	for skipList.level > 1 && skipList.head.next[skipList.level-1] == nil {
		skipList.level--
	}
}

func (skipList *SkipList[K, V]) Search(key K) (V, bool) {
	node := skipList.head
	for i := skipList.level - 1; i >= 0; i-- {
		for {
			if node.next[i] == nil {
				break
			}

			if skipList.compare(node.next[i].key, key) == Equal {
				return node.next[i].value, true
			}

			if skipList.compare(node.next[i].key, key) == Less {
				node = node.next[i]
				continue
			} else {
				break
			}
		}
	}
	var zero V
	return zero, false
}

func (skipList *SkipList[K, V]) PrintSkipList() {

	for i := skipList.maxLevel - 1; i >= 0; i-- {

		fmt.Println("level:", i)
		node := skipList.head.next[i]
		for {
			if node != nil {
				fmt.Printf("%v:%v %p ", node.key, node.value, node)
				node = node.next[i]
			} else {
				break
			}
		}
		fmt.Println("\n--------------------------------------------------------")
	}

	fmt.Println("Current MaxLevel:", skipList.level)
}
