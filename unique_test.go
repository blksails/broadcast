package broadcast

import (
	"fmt"
	"sync"
	"testing"
	"unique"
)

// TestData 实现 Uniquer 接口的测试结构体
type TestUniqueData struct {
	ID   int
	Name string
}

type TestUniquer struct {
	data TestUniqueData
}

func (t *TestUniquer) Unique() unique.Handle[int] {
	return unique.Make(t.data.ID)
}

func (t *TestUniquer) Value() TestUniqueData {
	return t.data
}

func TestUniqueBroadcast_Handle(t *testing.T) {
	b := &UniqueBroadcast[int, TestUniqueData]{}

	called := false
	handler := func(signal string, data TestUniqueData) error {
		called = true
		if signal != "test" || data.ID != 1 || data.Name != "test1" {
			t.Errorf("unexpected signal or data: got signal=%s, data=%+v", signal, data)
		}
		return nil
	}

	b.Handle(handler)
	b.Watch("test", &TestUniquer{data: TestUniqueData{ID: 1, Name: "test1"}})
	b.Broadcast("test")

	if !called {
		t.Error("handler was not called")
	}
}

func TestUniqueBroadcast_Watch(t *testing.T) {
	b := &UniqueBroadcast[int, TestUniqueData]{}

	// Test watching same signal with same ID
	data1 := &TestUniquer{data: TestUniqueData{ID: 1, Name: "test1"}}
	data2 := &TestUniquer{data: TestUniqueData{ID: 1, Name: "test2"}} // Same ID
	data3 := &TestUniquer{data: TestUniqueData{ID: 2, Name: "test3"}} // Different ID

	b.Watch("test", data1)
	b.Watch("test", data2) // Should not duplicate due to same ID
	b.Watch("test", data3)

	if len(b.listeners["test"]) != 2 {
		t.Errorf("expected 2 listeners, got %d", len(b.listeners["test"]))
	}
}

func TestUniqueBroadcast_Unwatch(t *testing.T) {
	b := &UniqueBroadcast[int, TestUniqueData]{}

	data1 := &TestUniquer{data: TestUniqueData{ID: 1, Name: "test1"}}
	data2 := &TestUniquer{data: TestUniqueData{ID: 2, Name: "test2"}}

	b.Watch("test", data1)
	b.Watch("test", data2)
	b.Unwatch("test", data1)

	if len(b.listeners["test"]) != 1 {
		t.Errorf("expected 1 listener after unwatch, got %d", len(b.listeners["test"]))
	}
}

func TestUniqueBroadcast_Concurrent(t *testing.T) {
	b := &UniqueBroadcast[int, TestUniqueData]{}
	var wg sync.WaitGroup
	counter := 0
	mutex := sync.Mutex{}

	handler := func(signal string, data TestUniqueData) error {
		mutex.Lock()
		counter++
		mutex.Unlock()
		return nil
	}

	b.Handle(handler)

	// Concurrent watching and broadcasting
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			b.Watch("test", &TestUniquer{data: TestUniqueData{ID: i, Name: "test"}})
		}(i)
		go func() {
			defer wg.Done()
			b.Broadcast("test")
		}()
	}

	wg.Wait()

	if counter == 0 {
		t.Error("handlers were not called in concurrent test")
	}
}

func TestUniqueBroadcast_MultipleHandlers(t *testing.T) {
	b := &UniqueBroadcast[int, TestUniqueData]{}
	calls := 0
	mutex := sync.Mutex{}

	// Register multiple handlers
	for i := 0; i < 3; i++ {
		b.Handle(func(signal string, data TestUniqueData) error {
			mutex.Lock()
			calls++
			mutex.Unlock()
			return nil
		})
	}

	data := &TestUniquer{data: TestUniqueData{ID: 1, Name: "test"}}
	b.Watch("test", data)
	b.Broadcast("test")

	if calls != 3 {
		t.Errorf("expected 3 handler calls, got %d", calls)
	}
}

func TestUniqueBroadcast_HasWatch(t *testing.T) {
	b := &UniqueBroadcast[int, TestUniqueData]{}

	// 测试空信号
	if b.HasWatch("test") {
		t.Error("empty signal should not have watchers")
	}

	// 添加监听器
	data := &TestUniquer{data: TestUniqueData{ID: 1, Name: "test"}}
	b.Watch("test", data)

	// 测试有监听器的信号
	if !b.HasWatch("test") {
		t.Error("signal should have watchers after Watch")
	}

	// 测试监听器数量
	if count := b.WatchCount("test"); count != 1 {
		t.Errorf("expected 1 watcher, got %d", count)
	}

	// 清除后测试
	b.Clean("test")
	if b.HasWatch("test") {
		t.Error("signal should not have watchers after Clean")
	}
}

func TestUniqueBroadcast_WatchCount(t *testing.T) {
	b := &UniqueBroadcast[int, TestUniqueData]{}

	// 测试空信号
	if count := b.WatchCount("test"); count != 0 {
		t.Errorf("empty signal should have 0 watchers, got %d", count)
	}

	// 添加多个不同ID的监听器
	for i := 0; i < 3; i++ {
		data := &TestUniquer{data: TestUniqueData{ID: i, Name: fmt.Sprintf("test%d", i)}}
		b.Watch("test", data)
	}

	// 验证数量
	if count := b.WatchCount("test"); count != 3 {
		t.Errorf("expected 3 watchers, got %d", count)
	}

	// 测试添加重复ID的情况
	data := &TestUniquer{data: TestUniqueData{ID: 0, Name: "duplicate"}}
	b.Watch("test", data)
	if count := b.WatchCount("test"); count != 3 {
		t.Errorf("watcher count should not increase for duplicate ID, got %d", count)
	}
}

func TestUniqueBroadcast_Clean(t *testing.T) {
	b := &UniqueBroadcast[int, TestUniqueData]{}

	// 添加多个监听器到不同的信号
	signals := []string{"test1", "test2", "test3"}
	for _, signal := range signals {
		for i := 0; i < 3; i++ {
			data := &TestUniquer{data: TestUniqueData{ID: i, Name: fmt.Sprintf("test_%s_%d", signal, i)}}
			b.Watch(signal, data)
		}
	}

	// 验证初始状态
	for _, signal := range signals {
		if len(b.listeners[signal]) != 3 {
			t.Errorf("expected 3 listeners for signal %s, got %d", signal, len(b.listeners[signal]))
		}
	}

	// 测试清除单个信号
	b.Clean("test1")
	if _, exists := b.listeners["test1"]; exists {
		t.Error("listeners for test1 should be removed")
	}
	if len(b.listeners["test2"]) != 3 {
		t.Error("listeners for test2 should remain unchanged")
	}

	// 测试 CleanAll
	b.CleanAll()
	if len(b.listeners) != 0 {
		t.Error("all listeners should be removed after CleanAll")
	}

	// 测试清除后重新添加
	data := &TestUniquer{data: TestUniqueData{ID: 1, Name: "test"}}
	b.Watch("test", data)
	if len(b.listeners["test"]) != 1 {
		t.Error("should be able to add listeners after cleaning")
	}
}

func TestUniqueBroadcast_Range(t *testing.T) {
	b := &UniqueBroadcast[int, TestUniqueData]{}

	// 准备测试数据
	expectedSignals := map[string]int{
		"signal1": 2,
		"signal2": 3,
		"signal3": 1,
	}

	for signal, count := range expectedSignals {
		for i := 0; i < count; i++ {
			data := &TestUniquer{data: TestUniqueData{ID: i, Name: fmt.Sprintf("test_%s_%d", signal, i)}}
			b.Watch(signal, data)
		}
	}

	// 测试遍历
	visitedSignals := make(map[string]int)
	b.Range(func(signal string, count int) bool {
		visitedSignals[signal] = count
		return true
	})

	// 验证结果
	if len(visitedSignals) != len(expectedSignals) {
		t.Errorf("expected %d signals, got %d", len(expectedSignals), len(visitedSignals))
	}

	for signal, expectedCount := range expectedSignals {
		if count, exists := visitedSignals[signal]; !exists || count != expectedCount {
			t.Errorf("signal %s: expected count %d, got %d", signal, expectedCount, count)
		}
	}

	// 测试提前终止
	visitCount := 0
	b.Range(func(signal string, count int) bool {
		visitCount++
		return visitCount < 2 // 只访问前两个信号
	})

	if visitCount != 2 {
		t.Errorf("Range should stop after visiting 2 signals, visited %d signals", visitCount)
	}

	// 测试空广播器
	emptyB := &UniqueBroadcast[int, TestUniqueData]{}
	emptyVisited := 0
	emptyB.Range(func(signal string, count int) bool {
		emptyVisited++
		return true
	})

	if emptyVisited != 0 {
		t.Errorf("empty broadcast should not visit any signals, visited %d", emptyVisited)
	}
}

// Benchmarks
func BenchmarkUniqueBroadcast_Watch(b *testing.B) {
	br := &UniqueBroadcast[int, TestUniqueData]{}
	data := &TestUniquer{data: TestUniqueData{ID: 1, Name: "test"}}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		br.Watch("test", data)
	}
}

func BenchmarkUniqueBroadcast_Unwatch(b *testing.B) {
	br := &UniqueBroadcast[int, TestUniqueData]{}
	data := &TestUniquer{data: TestUniqueData{ID: 1, Name: "test"}}
	br.Watch("test", data)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		br.Unwatch("test", data)
		br.Watch("test", data)
	}
}

func BenchmarkUniqueBroadcast_Broadcast(b *testing.B) {
	br := &UniqueBroadcast[int, TestUniqueData]{}
	handler := func(signal string, data TestUniqueData) error {
		return nil
	}
	br.Handle(handler)

	data1 := &TestUniquer{data: TestUniqueData{ID: 1, Name: "test1"}}
	data2 := &TestUniquer{data: TestUniqueData{ID: 2, Name: "test2"}}
	br.Watch("test", data1)
	br.Watch("test", data2)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		br.Broadcast("test")
	}
}

func BenchmarkUniqueBroadcast_ConcurrentBroadcast(b *testing.B) {
	br := &UniqueBroadcast[int, TestUniqueData]{}
	handler := func(signal string, data TestUniqueData) error {
		return nil
	}
	br.Handle(handler)

	for i := 0; i < 100; i++ {
		br.Watch("test", &TestUniquer{data: TestUniqueData{ID: i, Name: "test"}})
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			br.Broadcast("test")
		}
	})
}

func BenchmarkUniqueBroadcast_HasWatch(b *testing.B) {
	br := &UniqueBroadcast[int, TestUniqueData]{}
	data := &TestUniquer{data: TestUniqueData{ID: 1, Name: "test"}}
	br.Watch("test", data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		br.HasWatch("test")
	}
}

func BenchmarkUniqueBroadcast_WatchCount(b *testing.B) {
	br := &UniqueBroadcast[int, TestUniqueData]{}
	for i := 0; i < 100; i++ {
		data := &TestUniquer{data: TestUniqueData{ID: i, Name: "test"}}
		br.Watch("test", data)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		br.WatchCount("test")
	}
}

func BenchmarkUniqueBroadcast_Clean(b *testing.B) {
	br := &UniqueBroadcast[int, TestUniqueData]{}

	// 预先添加一些监听器
	for i := 0; i < 100; i++ {
		data := &TestUniquer{data: TestUniqueData{ID: i, Name: "test"}}
		br.Watch("test", data)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		br.Clean("test")
		// 重新添加监听器以准备下一次测试
		data := &TestUniquer{data: TestUniqueData{ID: 1, Name: "test"}}
		br.Watch("test", data)
	}
}

func BenchmarkUniqueBroadcast_CleanAll(b *testing.B) {
	br := &UniqueBroadcast[int, TestUniqueData]{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 添加一些监听器
		for j := 0; j < 10; j++ {
			data := &TestUniquer{data: TestUniqueData{ID: j, Name: "test"}}
			br.Watch(fmt.Sprintf("test%d", j), data)
		}
		br.CleanAll()
	}
}

func BenchmarkUniqueBroadcast_Range(b *testing.B) {
	br := &UniqueBroadcast[int, TestUniqueData]{}

	// 准备测试数据
	for i := 0; i < 100; i++ {
		signal := fmt.Sprintf("signal%d", i)
		data := &TestUniquer{data: TestUniqueData{ID: i, Name: "test"}}
		br.Watch(signal, data)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		br.Range(func(signal string, count int) bool {
			return true
		})
	}
}

func TestNewUnique(t *testing.T) {
	b := NewUnique[int, string]()

	if b.handlers == nil {
		t.Error("handlers should be initialized")
	}

	if b.listeners == nil {
		t.Error("listeners should be initialized")
	}

	if len(b.handlers) != 0 {
		t.Error("handlers should be empty")
	}

	if len(b.listeners) != 0 {
		t.Error("listeners should be empty")
	}
}
