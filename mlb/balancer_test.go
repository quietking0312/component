package mlb

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBalancer_BasicPick(t *testing.T) {
	b := NewBalancer()

	n1 := NewNode("node-1", "127.0.0.1:8001", 1, 100)
	n2 := NewNode("node-2", "127.0.0.1:8002", 1, 100)
	n3 := NewNode("node-3", "127.0.0.1:8003", 1, 100)

	b.Register(n1)
	b.Register(n2)
	b.Register(n3)

	require.NoError(t, b.Online("node-1"))
	require.NoError(t, b.Online("node-2"))
	require.NoError(t, b.Online("node-3"))

	// 同一用户多次分配应该路由到同一节点（一致性）
	userID := "user-10086"
	node1, err := b.Pick(userID)
	require.NoError(t, err)

	for i := 0; i < 100; i++ {
		node2, err := b.Pick(userID)
		require.NoError(t, err)
		assert.Equal(t, node1.ID, node2.ID, "同一用户应始终路由到同一节点")
	}
}

func TestBalancer_LoadBalance(t *testing.T) {
	b := NewBalancer()

	n1 := NewNode("node-1", "127.0.0.1:8001", 1, 1000)
	n2 := NewNode("node-2", "127.0.0.1:8002", 1, 1000)
	n3 := NewNode("node-3", "127.0.0.1:8003", 1, 1000)

	b.Register(n1)
	b.Register(n2)
	b.Register(n3)
	b.Online("node-1")
	b.Online("node-2")
	b.Online("node-3")

	// 分配大量用户，观察分布
	counts := map[string]int{
		"node-1": 0,
		"node-2": 0,
		"node-3": 0,
	}

	for i := 0; i < 3000; i++ {
		node, err := b.PickAndBind(fmt.Sprintf("user-%d", i))
		require.NoError(t, err)
		counts[node.ID]++
	}

	// 分布应该相对均匀（允许 30% 偏差）
	avg := 3000 / 3
	for id, c := range counts {
		assert.InDelta(t, avg, c, float64(avg)*0.30, "节点 %s 分配不均: %d", id, c)
	}

	t.Logf("分配分布: %v", counts)
}

func TestBalancer_NodeOffline_MinReassign(t *testing.T) {
	b := NewBalancer()

	n1 := NewNode("node-1", "127.0.0.1:8001", 1, 1000)
	n2 := NewNode("node-2", "127.0.0.1:8002", 1, 1000)
	n3 := NewNode("node-3", "127.0.0.1:8003", 1, 1000)

	b.Register(n1)
	b.Register(n2)
	b.Register(n3)
	b.Online("node-1")
	b.Online("node-2")
	b.Online("node-3")

	// 记录 3000 个用户的分配结果
	assignments := make(map[string]string)
	for i := 0; i < 3000; i++ {
		userID := fmt.Sprintf("user-%d", i)
		node, _ := b.Pick(userID)
		assignments[userID] = node.ID
	}

	// 下线 node-2
	b.Offline("node-2")

	// 统计需要重新分配的用户数
	reassigned := 0
	for i := 0; i < 3000; i++ {
		userID := fmt.Sprintf("user-%d", i)
		node, _ := b.Pick(userID)
		if assignments[userID] == "node-2" {
			// 原来在 node-2 的用户必须迁移
			assert.NotEqual(t, "node-2", node.ID)
		} else {
			// 不在 node-2 的用户应该尽量保持不动
			if assignments[userID] != node.ID {
				reassigned++
			}
		}
	}

	// 重新分配的用户比例应该很低（< 10%）
	reassignRate := float64(reassigned) / 3000.0
	t.Logf("下线 node-2 后，非必要重新分配用户: %d (%.2f%%)", reassigned, reassignRate*100)
	assert.Less(t, reassignRate, 0.10, "重新分配比例应低于 10%%")
}

func TestBalancer_NodeOffline_ReassignUniform(t *testing.T) {
	b := NewBalancer()

	n1 := NewNode("node-1", "127.0.0.1:8001", 1, 10000)
	n2 := NewNode("node-2", "127.0.0.1:8002", 1, 10000)
	n3 := NewNode("node-3", "127.0.0.1:8003", 1, 10000)
	n4 := NewNode("node-4", "127.0.0.1:8004", 1, 10000)

	for _, n := range []*Node{n1, n2, n3, n4} {
		b.Register(n)
		b.Online(n.ID)
	}

	// 下线 node-4，观察原来在 node-4 的用户被均匀分配到其他节点
	b.Offline("node-4")

	counts := map[string]int{
		"node-1": 0,
		"node-2": 0,
		"node-3": 0,
	}

	// 用大量用户测试分布
	for i := 0; i < 10000; i++ {
		userID := fmt.Sprintf("user-%d", i)
		node, _ := b.Pick(userID)
		if c, ok := counts[node.ID]; ok {
			counts[node.ID] = c + 1
		}
	}

	// 重新分配的用户应该均匀分散到剩余 3 个节点
	avg := counts["node-1"] + counts["node-2"] + counts["node-3"]
	if avg > 0 {
		avg /= 3
		for id, c := range counts {
			assert.InDelta(t, avg, c, float64(avg)*0.20, "重新分配到节点 %s 不均匀: %d", id, c)
		}
	}

	t.Logf("node-4 下线后用户分布: %v", counts)
}

func TestBalancer_Weight(t *testing.T) {
	b := NewBalancer()

	// node-1 权重为 1，node-2 权重为 3（性能是 node-1 的 3 倍）
	n1 := NewNode("node-1", "127.0.0.1:8001", 1, 1000)
	n2 := NewNode("node-2", "127.0.0.1:8002", 3, 3000)

	b.Register(n1)
	b.Register(n2)
	b.Online("node-1")
	b.Online("node-2")

	counts := map[string]int{
		"node-1": 0,
		"node-2": 0,
	}

	for i := 0; i < 4000; i++ {
		node, _ := b.PickAndBind(fmt.Sprintf("user-%d", i))
		counts[node.ID]++
	}

	// node-2 应该分配到约 3/4 的用户
	t.Logf("权重分配分布: %v", counts)
	total := counts["node-1"] + counts["node-2"]
	ratio := float64(counts["node-2"]) / float64(total)
	assert.InDelta(t, 0.75, ratio, 0.08, "权重为 3:1，分配比例应接近 75%%:25%%")
}

func TestBalancer_ActiveDecr_AffectsNewUsers(t *testing.T) {
	b := NewBalancer()

	n1 := NewNode("node-1", "127.0.0.1:8001", 1, 100)
	n2 := NewNode("node-2", "127.0.0.1:8002", 1, 100)

	b.Register(n1)
	b.Register(n2)
	b.Online("node-1")
	b.Online("node-2")

	// 让 node-1 满载，node-2 空载
	for i := 0; i < 100; i++ {
		n1.IncrActive()
	}

	// 新用户应该更倾向于分配到 node-2（负载低）
	node2Count := 0
	for i := 0; i < 100; i++ {
		node, _ := b.Pick(fmt.Sprintf("new-user-%d", i))
		if node.ID == "node-2" {
			node2Count++
		}
	}

	t.Logf("node-1 满载时，新用户分配到 node-2 的比例: %d/100", node2Count)
	assert.Greater(t, node2Count, 50, "node-1 满载时，大部分新用户应分配到 node-2")

	// 老用户从 node-1 下线
	for i := 0; i < 50; i++ {
		n1.DecrActive()
	}

	// 现在 node-1 负载降低，新用户应该更多分配到 node-1
	node1Count := 0
	for i := 100; i < 200; i++ {
		node, _ := b.Pick(fmt.Sprintf("new-user-%d", i))
		if node.ID == "node-1" {
			node1Count++
		}
	}

	t.Logf("node-1 释放 50%% 负载后，新用户分配到 node-1 的比例: %d/100", node1Count)
	assert.Greater(t, node1Count, 20, "node-1 负载降低后，应有更多新用户分配过来")
}

func TestBalancer_Unbind(t *testing.T) {
	b := NewBalancer()

	n1 := NewNode("node-1", "127.0.0.1:8001", 1, 100)
	b.Register(n1)
	b.Online("node-1")

	n1.IncrActive()
	n1.IncrActive()
	assert.Equal(t, int64(2), n1.ActiveCount())

	b.Unbind("node-1")
	assert.Equal(t, int64(1), n1.ActiveCount())

	b.Unbind("node-1")
	assert.Equal(t, int64(0), n1.ActiveCount())

	// 不会减到负数
	b.Unbind("node-1")
	assert.Equal(t, int64(0), n1.ActiveCount())
}

func TestBalancer_Stats(t *testing.T) {
	b := NewBalancer()

	n1 := NewNode("node-1", "127.0.0.1:8001", 1, 100)
	n2 := NewNode("node-2", "127.0.0.1:8002", 1, 100)

	b.Register(n1)
	b.Register(n2)
	b.Online("node-1")
	b.Online("node-2")

	for i := 0; i < 50; i++ {
		n1.IncrActive()
	}
	for i := 0; i < 25; i++ {
		n2.IncrActive()
	}

	stats := b.GetStats()
	assert.Equal(t, 2, stats.TotalNodes)
	assert.Equal(t, 2, stats.OnlineNodes)
	assert.Equal(t, int64(75), stats.TotalActive)
	assert.InDelta(t, 0.375, stats.AvgLoadRate, 0.01)
}

func TestBalancer_EmptyRing(t *testing.T) {
	b := NewBalancer()
	_, err := b.Pick("user-1")
	assert.Equal(t, ErrNoAvailableNode, err)
}

func TestBalancer_NodeNotFound(t *testing.T) {
	b := NewBalancer()
	err := b.Online("not-exist")
	assert.Equal(t, ErrNodeNotFound, err)
}

func TestNode_LoadRate(t *testing.T) {
	n := NewNode("n1", "", 1, 100)
	assert.Equal(t, 0.0, n.LoadRate())

	n.IncrActive()
	n.IncrActive()
	assert.Equal(t, 0.02, n.LoadRate())

	for i := 0; i < 98; i++ {
		n.IncrActive()
	}
	assert.Equal(t, 1.0, n.LoadRate())
	assert.True(t, n.IsOverloaded())
}

func TestNode_EffectiveWeight(t *testing.T) {
	n := NewNode("n1", "", 10, 100)

	// 空载时有效权重接近基础权重
	assert.InDelta(t, 10.0, n.EffectiveWeight(), 0.01)

	// 50% 负载
	for i := 0; i < 50; i++ {
		n.IncrActive()
	}
	// 10 * (1 - 0.5^2) = 7.5
	assert.InDelta(t, 7.5, n.EffectiveWeight(), 0.01)

	// 满载
	for i := 0; i < 50; i++ {
		n.IncrActive()
	}
	assert.InDelta(t, 0.0, n.EffectiveWeight(), 0.01)
}

func BenchmarkPick(b *testing.B) {
	bal := NewBalancer()

	for i := 0; i < 10; i++ {
		n := NewNode(fmt.Sprintf("node-%d", i), fmt.Sprintf("127.0.0.1:800%d", i), 1, 10000)
		bal.Register(n)
		bal.Online(n.ID)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = bal.Pick(fmt.Sprintf("user-%d", i))
	}
}
