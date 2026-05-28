package mlb

import (
	"fmt"
	"log"
)

func ExampleBalancer() {
	balancer := NewBalancer()

	// 注册 3 个逻辑服务器
	balancer.Register(NewNode("logic-1", "10.0.0.1:8001", 1, 1000))
	balancer.Register(NewNode("logic-2", "10.0.0.2:8002", 1, 1000))
	balancer.Register(NewNode("logic-3", "10.0.0.3:8003", 2, 2000))

	// 全部上线
	_ = balancer.Online("logic-1")
	_ = balancer.Online("logic-2")
	_ = balancer.Online("logic-3")

	// 分配新用户
	node, err := balancer.PickAndBind("user-1001")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("分配节点:", node.ID)

	// 用户下线
	_ = balancer.Unbind(node.ID)

	// 节点下线维护
	_ = balancer.Offline("logic-2")

	// 输出负载统计
	stats := balancer.GetStats()
	fmt.Printf("在线节点: %d\n", stats.OnlineNodes)

	// Output:
	// 分配节点: logic-3
	// 在线节点: 2
}

func ExampleBalancer_offline() {
	balancer := NewBalancer()

	balancer.Register(NewNode("logic-1", "10.0.0.1:8001", 1, 10000))
	balancer.Register(NewNode("logic-2", "10.0.0.2:8002", 1, 10000))
	balancer.Register(NewNode("logic-3", "10.0.0.3:8003", 1, 10000))

	_ = balancer.Online("logic-1")
	_ = balancer.Online("logic-2")
	_ = balancer.Online("logic-3")

	// 记录 1000 个用户的初始分配
	assignments := make(map[string]string)
	for i := 0; i < 1000; i++ {
		userID := fmt.Sprintf("user-%d", i)
		node, _ := balancer.Pick(userID)
		assignments[userID] = node.ID
	}

	// 下线 logic-2，观察重新分配情况
	_ = balancer.Offline("logic-2")

	reassigned := 0
	for i := 0; i < 1000; i++ {
		userID := fmt.Sprintf("user-%d", i)
		node, _ := balancer.Pick(userID)
		if assignments[userID] != "logic-2" && assignments[userID] != node.ID {
			reassigned++
		}
	}

	fmt.Printf("非必要重新分配: %d\n", reassigned)
	// Output:
	// 非必要重新分配: 0
}

func ExampleNode_LoadRate() {
	node := NewNode("logic-1", "10.0.0.1:8001", 1, 100)

	fmt.Printf("初始负载率: %.2f\n", node.LoadRate())

	// 模拟 50 个用户连接
	for i := 0; i < 50; i++ {
		node.IncrActive()
	}
	fmt.Printf("50%% 负载率: %.2f\n", node.LoadRate())

	// 用户下线
	for i := 0; i < 50; i++ {
		node.DecrActive()
	}
	fmt.Printf("全部下线后负载率: %.2f\n", node.LoadRate())

	// Output:
	// 初始负载率: 0.00
	// 50% 负载率: 0.50
	// 全部下线后负载率: 0.00
}
