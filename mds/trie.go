package mds

type trieNode struct {
	children map[rune]*trieNode
	fail     *trieNode // AC: 失配指针
	output   *trieNode // AC: 输出链，指向 fail 链上最近的模式终点
	pattern  string    // 非空表示此节点是某个模式词的终点（同时充当 isEnd）
	count    int       // 经过此节点的单词数，用于删除时判断是否可以裁剪
}

func newTrieNode() *trieNode {
	return &trieNode{children: make(map[rune]*trieNode)}
}

type Trie struct {
	root   *trieNode
	dirty  bool            // Insert/Delete 后置 true，Build 后置 false
	except func(rune) bool // 返回 true 的字符在 Match 时跳过（不参与匹配，但保留在结果坐标中）
}

func NewTrie() *Trie {
	return &Trie{
		root:   newTrieNode(),
		dirty:  true,
		except: func(rune) bool { return false },
	}
}

func (t *Trie) SetExcept(except func(rune) bool) {
	t.except = except
}

func (t *Trie) Insert(word string) {
	node := t.root
	for _, c := range word {
		if _, ok := node.children[c]; !ok {
			node.children[c] = newTrieNode()
		}
		node = node.children[c]
		node.count++
	}
	node.pattern = word
	t.dirty = true
}

// Search 返回 word 是否完整存在于前缀树中
func (t *Trie) Search(word string) bool {
	node := t.find(word)
	return node != nil && node.pattern != ""
}

// StartsWith 返回是否存在以 prefix 开头的单词
func (t *Trie) StartsWith(prefix string) bool {
	return t.find(prefix) != nil
}

// Delete 删除 word，返回 word 是否存在（不存在则不做任何操作）
func (t *Trie) Delete(word string) bool {
	if !t.Search(word) {
		return false
	}
	node := t.root
	runes := []rune(word)
	for i, c := range runes {
		child := node.children[c]
		child.count--
		if child.count == 0 {
			delete(node.children, c)
			_ = runes[i:]
			t.dirty = true
			return true
		}
		node = child
	}
	node.pattern = ""
	t.dirty = true
	return true
}

// WordsWithPrefix 返回所有以 prefix 开头的单词
func (t *Trie) WordsWithPrefix(prefix string) []string {
	node := t.find(prefix)
	if node == nil {
		return nil
	}
	result := make([]string, 0)
	t.dfs(node, []rune(prefix), &result)
	return result
}

// Build 构建 AC 自动机失配指针，Insert/Delete 后需调用一次
func (t *Trie) Build() {
	queue := make([]*trieNode, 0, 64)
	for _, child := range t.root.children {
		child.fail = t.root
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
				child.fail = t.root
			}
			if child.fail.pattern != "" {
				child.output = child.fail
			} else {
				child.output = child.fail.output
			}
			queue = append(queue, child)
		}
	}
	t.dirty = false
}

// Match 返回文本中所有命中模式词的位置 [起始下标, 结束下标]（闭区间，按 rune 计）
// except 字符会被跳过不参与匹配，但位置坐标仍按原文计算。
// 若 Insert/Delete 后未调用 Build，会自动触发一次 Build。
func (t *Trie) Match(text string) [][2]int {
	if t.dirty {
		t.Build()
	}
	runes := []rune(text)
	result := make([][2]int, 0)
	node := t.root

	// positions[k] = 第 k 个有效字符（非 except）在原文中的 rune 下标
	positions := make([]int, 0, len(runes))

	for i, c := range runes {
		if t.except(c) {
			continue
		}
		positions = append(positions, i)
		k := len(positions) - 1

		for node != t.root {
			if _, ok := node.children[c]; ok {
				break
			}
			node = node.fail
		}
		if next, ok := node.children[c]; ok {
			node = next
		}

		collect := func(n *trieNode) {
			wlen := len([]rune(n.pattern))
			result = append(result, [2]int{positions[k-wlen+1], i})
		}
		if node.pattern != "" {
			collect(node)
		}
		for out := node.output; out != nil; out = out.output {
			collect(out)
		}
	}
	return result
}

func (t *Trie) find(s string) *trieNode {
	node := t.root
	for _, c := range s {
		child, ok := node.children[c]
		if !ok {
			return nil
		}
		node = child
	}
	return node
}

func (t *Trie) dfs(node *trieNode, path []rune, result *[]string) {
	if node.pattern != "" {
		*result = append(*result, string(path))
	}
	for c, child := range node.children {
		t.dfs(child, append(path, c), result)
	}
}
