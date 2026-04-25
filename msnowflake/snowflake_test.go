package msnowflake

import (
	"sync"
	"testing"
	"time"
)

func TestNewGenerator(t *testing.T) {
	tests := []struct {
		name     string
		workerID int64
		wantErr  bool
	}{
		{"valid worker 0", 0, false},
		{"valid worker 1", 1, false},
		{"valid worker max", maxWorkerID, false},
		{"invalid worker negative", -1, true},
		{"invalid worker too large", maxWorkerID + 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := NewGenerator(tt.workerID)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGenerator() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && g == nil {
				t.Error("NewGenerator() returned nil generator without error")
			}
		})
	}
}

func TestNewGeneratorWithEpoch(t *testing.T) {
	now := time.Now().UnixMilli()

	tests := []struct {
		name    string
		worker  int64
		epoch   int64
		wantErr bool
	}{
		{"valid", 0, defaultEpoch, false},
		{"negative epoch", 0, -1, true},
		{"future epoch", 0, now + 10000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewGeneratorWithEpoch(tt.worker, tt.epoch)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGeneratorWithEpoch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerator_NextID(t *testing.T) {
	g, err := NewGenerator(1)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	// 生成一些ID
	idSet := make(map[int64]bool)
	count := 10000

	for i := 0; i < count; i++ {
		id := g.NextID()
		if idSet[id] {
			t.Errorf("duplicate ID generated: %d", id)
		}
		idSet[id] = true
	}

	if len(idSet) != count {
		t.Errorf("expected %d unique IDs, got %d", count, len(idSet))
	}
}

func TestGenerator_NextID_Uniqueness(t *testing.T) {
	const numGoroutines = 100
	const idsPerGoroutine = 1000

	g, err := NewGenerator(1)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	var wg sync.WaitGroup
	idChan := make(chan int64, numGoroutines*idsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerGoroutine; j++ {
				idChan <- g.NextID()
			}
		}()
	}

	wg.Wait()
	close(idChan)

	idSet := make(map[int64]bool)
	for id := range idChan {
		if idSet[id] {
			t.Errorf("duplicate ID generated: %d", id)
		}
		idSet[id] = true
	}

	expectedCount := numGoroutines * idsPerGoroutine
	if len(idSet) != expectedCount {
		t.Errorf("expected %d unique IDs, got %d", expectedCount, len(idSet))
	}
}

func TestGenerator_NextIDs(t *testing.T) {
	g, err := NewGenerator(1)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	ids := g.NextIDs(100)
	if len(ids) != 100 {
		t.Errorf("expected 100 IDs, got %d", len(ids))
	}

	idSet := make(map[int64]bool)
	for _, id := range ids {
		if idSet[id] {
			t.Errorf("duplicate ID in batch: %d", id)
		}
		idSet[id] = true
	}
}

func TestParseID(t *testing.T) {
	g, err := NewGenerator(5)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	id := g.NextID()
	timestamp, workerID, sequence := ParseID(id, defaultEpoch)

	if workerID != 5 {
		t.Errorf("expected workerID 5, got %d", workerID)
	}

	if timestamp < defaultEpoch {
		t.Errorf("timestamp should be >= epoch, got %d", timestamp)
	}

	if sequence < 0 || sequence > maxSequence {
		t.Errorf("sequence out of range: %d", sequence)
	}
}

func TestParseIDWithDefaultEpoch(t *testing.T) {
	g, err := NewGenerator(10)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	id := g.NextID()
	timestamp, workerID, sequence := ParseIDWithDefaultEpoch(id)

	if workerID != 10 {
		t.Errorf("expected workerID 10, got %d", workerID)
	}

	if timestamp < defaultEpoch {
		t.Errorf("timestamp should be >= epoch, got %d", timestamp)
	}

	if sequence < 0 || sequence > maxSequence {
		t.Errorf("sequence out of range: %d", sequence)
	}
}

func TestGenerator_GetWorkerID(t *testing.T) {
	g, err := NewGenerator(42)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	if g.GetWorkerID() != 42 {
		t.Errorf("expected workerID 42, got %d", g.GetWorkerID())
	}
}

func TestGenerator_GetEpoch(t *testing.T) {
	customEpoch := int64(1609459200000) // 2021-01-01
	g, err := NewGeneratorWithEpoch(1, customEpoch)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	if g.GetEpoch() != customEpoch {
		t.Errorf("expected epoch %d, got %d", customEpoch, g.GetEpoch())
	}
}

func TestMaxWorkerID(t *testing.T) {
	if MaxWorkerID() != 1023 {
		t.Errorf("expected max worker ID 1023, got %d", MaxWorkerID())
	}
}

func TestMaxSequence(t *testing.T) {
	if MaxSequence() != 4095 {
		t.Errorf("expected max sequence 4095, got %d", MaxSequence())
	}
}

func TestInitDefault(t *testing.T) {
	// 重置 once
	defaultGenerator = nil
	defaultGeneratorOnce = sync.Once{}

	err := InitDefault(1)
	if err != nil {
		t.Errorf("InitDefault() error = %v", err)
	}

	// 重复初始化应该无影响
	err = InitDefault(2)
	if err != nil {
		t.Errorf("second InitDefault() error = %v", err)
	}

	// 应该仍然使用第一次的 workerID
	if Default().GetWorkerID() != 1 {
		t.Errorf("expected workerID 1, got %d", Default().GetWorkerID())
	}

	// 生成ID不应该panic
	id := Generate()
	if id == 0 {
		t.Error("Generate() returned 0")
	}
}

func TestDefault_NotInitialized(t *testing.T) {
	// 保存当前状态
	savedGen := defaultGenerator
	defaultGenerator = nil

	defer func() {
		defaultGenerator = savedGen
	}()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Default() should panic when not initialized")
		}
	}()

	Default()
}

func TestGenerates(t *testing.T) {
	// 重置 once
	defaultGenerator = nil
	defaultGeneratorOnce = sync.Once{}

	err := InitDefault(1)
	if err != nil {
		t.Fatalf("InitDefault failed: %v", err)
	}

	ids := Generates(50)
	if len(ids) != 50 {
		t.Errorf("expected 50 IDs, got %d", len(ids))
	}

	idSet := make(map[int64]bool)
	for _, id := range ids {
		if idSet[id] {
			t.Errorf("duplicate ID: %d", id)
		}
		idSet[id] = true
	}
}

func BenchmarkGenerator_NextID(b *testing.B) {
	g, _ := NewGenerator(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.NextID()
	}
}

func BenchmarkGenerator_NextID_Parallel(b *testing.B) {
	g, _ := NewGenerator(1)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			g.NextID()
		}
	})
}
