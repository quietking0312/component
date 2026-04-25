package msnowflake

import (
	"fmt"
	"time"
)

// ExampleNewGenerator 演示如何创建雪花ID生成器
func ExampleNewGenerator() {
	// 创建生成器，指定机器ID为 1
	g, err := NewGenerator(1)
	if err != nil {
		panic(err)
	}

	// 生成ID
	id := g.NextID()
	fmt.Printf("Generated ID: %d\n", id)
}

// ExampleNewGeneratorWithEpoch 演示如何创建自定义起始时间的生成器
func ExampleNewGeneratorWithEpoch() {
	// 自定义起始时间：2024-01-01 00:00:00 UTC
	customEpoch := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	g, err := NewGeneratorWithEpoch(1, customEpoch)
	if err != nil {
		panic(err)
	}

	id := g.NextID()
	fmt.Printf("Generated ID with custom epoch: %d\n", id)
}

// ExampleGenerator_NextID 演示如何生成单个ID
func ExampleGenerator_NextID() {
	g, _ := NewGenerator(1)

	// 生成单个ID
	id := g.NextID()
	fmt.Printf("ID: %d\n", id)
}

// ExampleGenerator_NextIDs 演示如何批量生成ID
func ExampleGenerator_NextIDs() {
	g, _ := NewGenerator(1)

	// 批量生成10个ID
	ids := g.NextIDs(10)
	fmt.Printf("Generated %d IDs\n", len(ids))
	fmt.Printf("First ID: %d\n", ids[0])
}

// ExampleParseIDWithDefaultEpoch 演示如何解析ID
func ExampleParseIDWithDefaultEpoch() {
	g, _ := NewGenerator(5)

	// 生成ID
	id := g.NextID()

	// 解析ID
	timestamp, workerID, sequence := ParseIDWithDefaultEpoch(id)

	fmt.Printf("ID: %d\n", id)
	fmt.Printf("Timestamp: %d (%s)\n", timestamp, time.UnixMilli(timestamp).Format("2006-01-02 15:04:05"))
	fmt.Printf("Worker ID: %d\n", workerID)
	fmt.Printf("Sequence: %d\n", sequence)
}

// ExampleInitDefault 演示如何使用全局默认生成器
func ExampleInitDefault() {
	// 初始化全局默认生成器（只需执行一次，通常在程序启动时）
	err := InitDefault(1)
	if err != nil {
		panic(err)
	}

	// 生成单个ID
	id := Generate()
	fmt.Printf("Generated ID: %d\n", id)

	// 批量生成ID
	ids := Generates(5)
	fmt.Printf("Generated %d IDs\n", len(ids))
}

// Example_multiWorker 演示多机器ID的用法
func Example_multiWorker() {
	// 模拟两个不同的服务器节点
	worker1, _ := NewGenerator(1)
	worker2, _ := NewGenerator(2)

	// 每个节点生成ID
	id1 := worker1.NextID()
	id2 := worker2.NextID()

	// 解析查看机器ID
	_, workerID1, _ := ParseIDWithDefaultEpoch(id1)
	_, workerID2, _ := ParseIDWithDefaultEpoch(id2)

	fmt.Printf("Worker 1 generated ID %d (workerID=%d)\n", id1, workerID1)
	fmt.Printf("Worker 2 generated ID %d (workerID=%d)\n", id2, workerID2)
}
