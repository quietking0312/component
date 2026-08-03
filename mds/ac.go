package mds

// AC 自动机（Aho-Corasick Automaton）
// 多模式字符串匹配，时间复杂度 O(n + m + z)
// n: 文本长度，m: 所有模式总长度，z: 匹配数量

type acNode struct {
	children map[rune]*acNode
	fail     *acNode // 失配指针：最长真后缀对应的 trie 节点
	output   *acNode // 输出链：fail 链上最近的模式终点
	word     string  // 非空表示此节点是某个模式词的终点
}

func newACNode() *acNode {
	return &acNode{children: make(map[rune]*acNode)}
}

type AC struct {
	root *acNode
}

func NewAC() *AC {
	return &AC{root: newACNode()}
}

func (a *AC) AddWord(word string) {
	node := a.root
	for _, c := range word {
		if _, ok := node.children[c]; !ok {
			node.children[c] = newACNode()
		}
		node = node.children[c]
	}
	node.word = word
}

// Build 所有 AddWord 完成后调用，构建失配指针与输出链
func (a *AC) Build() {
	queue := make([]*acNode, 0, 64)
	for _, child := range a.root.children {
		child.fail = a.root
		queue = append(queue, child)
	}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for c, child := range curr.children {
			fail := curr.fail
			for fail != nil {
				if next, ok := fail.children[c]; ok {
					child.fail = next
					break
				}
				fail = fail.fail
			}
			if child.fail == nil {
				child.fail = a.root
			}
			if child.fail.word != "" {
				child.output = child.fail
			} else {
				child.output = child.fail.output
			}
			queue = append(queue, child)
		}
	}
}

// Search 返回文本中所有匹配模式的位置 [起始下标, 结束下标]（闭区间，按 rune 计）
func (a *AC) Search(text string) [][2]int {
	runes := []rune(text)
	result := make([][2]int, 0)
	node := a.root
	for i, c := range runes {
		for node != a.root {
			if _, ok := node.children[c]; ok {
				break
			}
			node = node.fail
		}
		if next, ok := node.children[c]; ok {
			node = next
		}
		if node.word != "" {
			start := i - len([]rune(node.word)) + 1
			result = append(result, [2]int{start, i})
		}
		for out := node.output; out != nil; out = out.output {
			start := i - len([]rune(out.word)) + 1
			result = append(result, [2]int{start, i})
		}
	}
	return result
}
