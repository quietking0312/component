package mtool

import (
	"fmt"
	"math/rand"
)

type SkipListNode struct {
	next  []*SkipListNode // 每一层的下一跳指针
	key   int64
	value any
}

func newNode(key int64, val any, lv int) *SkipListNode {
	return &SkipListNode{next: make([]*SkipListNode, lv), key: key, value: val}
}

type SkipList struct {
	head     *SkipListNode // 头节点
	level    int           // 当前最高层数
	maxLevel int           // 最大层数
	p        float64       // 随机提升概率 1 百分百提升
}

func NewSkipList() *SkipList {
	return &SkipList{level: 1, maxLevel: 8, p: 0.5, head: newNode(0, nil, 8)}
}

func (skipList *SkipList) Insert(key int64, value any) {
	update := make(map[int]*SkipListNode)
	curr := skipList.head

	for i := skipList.level - 1; i >= 0; i-- {
		for curr.next[i] != nil && curr.next[i].key < key {
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

func (skipList *SkipList) RandomLevel() int {
	var level int = 1
	for rand.Float64() < skipList.p && level < skipList.maxLevel {
		level++
	}
	return level
}

func (skipList *SkipList) Remove(key int64) {

	update := make(map[int]*SkipListNode)
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
			if curr.next[i].key < key {
				curr = curr.next[i]
				continue
			} else {
				break
			}
		}
	}

	for i, v := range update {
		if v == skipList.head {
			skipList.level--
		}
		v.next[i] = v.next[i].next[i]
	}
}

func (skipList *SkipList) Search(key int64) (any, bool) {
	node := skipList.head
	for i := skipList.level - 1; i >= 0; i-- {
		for {
			if node.next[i] == nil {
				break
			}

			if node.next[i].key == key {
				return node.next[i].value, true
			}

			if node.next[i].key < key {
				node = node.next[i]
				continue
			} else {
				break
			}
		}
	}
	return nil, false
}

func (skipList *SkipList) PrintSkipList() {

	for i := skipList.maxLevel - 1; i >= 0; i-- {

		fmt.Println("level:", i)
		node := skipList.head.next[i]
		for {
			if node != nil {
				fmt.Printf("%d:%v %p ", node.key, node.value, node)
				node = node.next[i]
			} else {
				break
			}
		}
		fmt.Println("\n--------------------------------------------------------")
	}

	fmt.Println("Current MaxLevel:", skipList.level)
}
