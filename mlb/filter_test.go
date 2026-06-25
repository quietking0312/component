package mlb

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNode_Tags(t *testing.T) {
	n := NewNode("n1", "127.0.0.1:8001", 1, 100, WithTags(map[string]string{
		"version": "v2",
		"region":  "ap-southeast-1",
	}))

	assert.True(t, n.HasTag("version", "v2"))
	assert.False(t, n.HasTag("version", "v1"))
	assert.False(t, n.HasTag("missing", "v1"))

	v, ok := n.Tag("region")
	assert.True(t, ok)
	assert.Equal(t, "ap-southeast-1", v)

	_, ok = n.Tag("missing")
	assert.False(t, ok)
}

func TestWithTags_Copy(t *testing.T) {
	src := map[string]string{"version": "v1"}
	n := NewNode("n1", "127.0.0.1:8001", 1, 100, WithTags(src))

	// 修改源 map 不应影响节点
	src["version"] = "v2"
	assert.True(t, n.HasTag("version", "v1"))
}

func TestBalancer_Pick_WithFilter(t *testing.T) {
	b := NewBalancer()

	n1 := NewNode("node-v1-a", "127.0.0.1:8001", 1, 1000, WithTags(map[string]string{"version": "v1"}))
	n2 := NewNode("node-v1-b", "127.0.0.1:8002", 1, 1000, WithTags(map[string]string{"version": "v1"}))
	n3 := NewNode("node-v2-a", "127.0.0.1:8003", 1, 1000, WithTags(map[string]string{"version": "v2"}))
	n4 := NewNode("node-v2-b", "127.0.0.1:8004", 1, 1000, WithTags(map[string]string{"version": "v2"}))

	for _, n := range []*Node{n1, n2, n3, n4} {
		b.Register(n)
		require.NoError(t, b.Online(n.ID))
	}

	// 只选择 v2 节点
	filterV2 := WithFilter(func(n *Node) bool {
		return n.HasTag("version", "v2")
	})

	v2IDs := map[string]int{
		n3.ID: 0,
		n4.ID: 0,
	}
	for i := 0; i < 1000; i++ {
		node, err := b.Pick(fmt.Sprintf("user-%d", i), filterV2)
		require.NoError(t, err)
		assert.Contains(t, []string{n3.ID, n4.ID}, node.ID, "必须只分配到 v2 节点")
		v2IDs[node.ID]++
	}

	// 两个 v2 节点都应被分配到
	assert.Greater(t, v2IDs[n3.ID], 0, "v2 节点 node-v2-a 应被分配")
	assert.Greater(t, v2IDs[n4.ID], 0, "v2 节点 node-v2-b 应被分配")

	// 同一用户多次 Pick 在节点集合不变时应保持一致
	userID := "user-stable"
	node1, err := b.Pick(userID, filterV2)
	require.NoError(t, err)
	for i := 0; i < 100; i++ {
		node2, err := b.Pick(userID, filterV2)
		require.NoError(t, err)
		assert.Equal(t, node1.ID, node2.ID, "同一用户应始终路由到同一 v2 节点")
	}
}

func TestBalancer_Pick_WithFilter_NoMatch(t *testing.T) {
	b := NewBalancer()
	n1 := NewNode("node-1", "127.0.0.1:8001", 1, 100, WithTags(map[string]string{"version": "v1"}))
	b.Register(n1)
	require.NoError(t, b.Online(n1.ID))

	_, err := b.Pick("user-1", WithFilter(func(n *Node) bool {
		return n.HasTag("version", "v2")
	}))
	assert.Equal(t, ErrNoAvailableNode, err)
}

func TestBalancer_Pick_WithFilter_OverloadedFallback(t *testing.T) {
	b := NewBalancer()

	n1 := NewNode("node-v2-1", "127.0.0.1:8001", 1, 10, WithTags(map[string]string{"version": "v2"}))
	n2 := NewNode("node-v2-2", "127.0.0.1:8002", 1, 10, WithTags(map[string]string{"version": "v2"}))
	n3 := NewNode("node-v1", "127.0.0.1:8003", 1, 10, WithTags(map[string]string{"version": "v1"}))

	for _, n := range []*Node{n1, n2, n3} {
		b.Register(n)
		require.NoError(t, b.Online(n.ID))
	}

	// 让 v2 节点全部满载
	for i := 0; i < 10; i++ {
		n1.IncrActive()
		n2.IncrActive()
	}

	filterV2 := WithFilter(func(n *Node) bool {
		return n.HasTag("version", "v2")
	})

	// 即使全部过载，仍应返回负载最低的 v2 节点兜底
	node, err := b.Pick("new-user", filterV2)
	require.NoError(t, err)
	assert.True(t, node.HasTag("version", "v2"))
}

func TestBalancer_PickAndBind_WithFilter(t *testing.T) {
	b := NewBalancer()
	n1 := NewNode("node-v1", "127.0.0.1:8001", 1, 100, WithTags(map[string]string{"version": "v1"}))
	n2 := NewNode("node-v2", "127.0.0.1:8002", 1, 100, WithTags(map[string]string{"version": "v2"}))

	b.Register(n1)
	b.Register(n2)
	require.NoError(t, b.Online(n1.ID))
	require.NoError(t, b.Online(n2.ID))

	node, err := b.PickAndBind("user-1", WithFilter(func(n *Node) bool {
		return n.HasTag("version", "v2")
	}))
	require.NoError(t, err)
	assert.Equal(t, n2.ID, node.ID)
	assert.Equal(t, int64(1), n2.ActiveCount())
	assert.Equal(t, int64(0), n1.ActiveCount())
}

func BenchmarkPick_WithFilter(b *testing.B) {
	bal := NewBalancer()

	for i := 0; i < 10; i++ {
		version := "v1"
		if i%2 == 0 {
			version = "v2"
		}
		n := NewNode(
			fmt.Sprintf("node-%d", i),
			fmt.Sprintf("127.0.0.1:800%d", i),
			1, 10000,
			WithTags(map[string]string{"version": version}),
		)
		bal.Register(n)
		bal.Online(n.ID)
	}

	filter := WithFilter(func(n *Node) bool {
		return n.HasTag("version", "v2")
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = bal.Pick(fmt.Sprintf("user-%d", i), filter)
	}
}
