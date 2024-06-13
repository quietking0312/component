package mtool

type MapTreeNode struct {
	Children map[rune]*MapTreeNode
	IsEnd    bool
}

func NewMapTreeNode() *MapTreeNode {
	return &MapTreeNode{
		Children: make(map[rune]*MapTreeNode),
		IsEnd:    false,
	}
}

type MapTree struct {
	root   *MapTreeNode
	except func(s rune) bool
}

func NewMapTree() *MapTree {
	return &MapTree{
		root: NewMapTreeNode(),
		except: func(s rune) bool {
			return false
		},
	}
}

func (t *MapTree) SetExcept(except func(s rune) bool) {
	t.except = except
}

func (t *MapTree) AddWord(word string) {
	node := t.root
	for _, c := range word {
		if _, ok := node.Children[c]; !ok {
			node.Children[c] = NewMapTreeNode()
		}
		node = node.Children[c]
	}
	node.IsEnd = true
}

func (t *MapTree) Load(text string) [][2]int {
	runes := []rune(text)
	reply := make([][2]int, 0)
	for i := 0; i < len(runes); i++ {
		node := t.root
		for j := i; j < len(runes); j++ {
			if t.except(runes[j]) {
				continue
			}
			if _, ok := node.Children[runes[j]]; ok {
				node = node.Children[runes[j]]
				if node.IsEnd {
					reply = append(reply, [2]int{i, j})
					i = j
					node = t.root
					break
				}
			} else {
				break
			}
		}
	}
	return reply
}
