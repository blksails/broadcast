package broadcast

import (
	"fmt"
	"sync"
	"testing"
)

func TestBroadcast_Handle(t *testing.T) {
	b := New[string]()

	called := false
	handler := func(signal string, data string, metadata map[string]interface{}) error {
		called = true
		if signal != "test" || data != "data" {
			t.Errorf("expected signal 'test' and data 'data', got signal '%s' and data '%s'", signal, data)
		}
		return nil
	}

	b.Handle(handler)
	b.Watch("test", "data")
	b.Broadcast("test", nil)

	if !called {
		t.Error("handler was not called")
	}
}

func TestBroadcast_Watch(t *testing.T) {
	b := New[string]()

	// Test watching same signal multiple times
	b.Watch("test", "data1")
	b.Watch("test", "data1") // Should not duplicate
	b.Watch("test", "data2")

	if len(b.listeners["test"]) != 2 {
		t.Errorf("expected 2 listeners, got %d", len(b.listeners["test"]))
	}
}

func TestBroadcast_Unwatch(t *testing.T) {
	b := New[string]()

	b.Watch("test", "data1")
	b.Watch("test", "data2")
	b.Unwatch("test", "data1")

	if len(b.listeners["test"]) != 1 {
		t.Errorf("expected 1 listener after unwatch, got %d", len(b.listeners["test"]))
	}
}

func TestBroadcast_Concurrent(t *testing.T) {
	b := New[int]()
	var wg sync.WaitGroup
	counter := 0
	mutex := sync.Mutex{}

	handler := func(signal string, data int, metadata map[string]interface{}) error {
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
			b.Watch("test", i)
		}(i)
		go func() {
			defer wg.Done()
			b.Broadcast("test", nil)
		}()
	}

	wg.Wait()

	if counter == 0 {
		t.Error("handlers were not called in concurrent test")
	}
}

// Benchmarks

func BenchmarkBroadcast_Watch(b *testing.B) {
	br := New[string]()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		br.Watch("test", "data")
	}
}

func BenchmarkBroadcast_Unwatch(b *testing.B) {
	br := New[string]()
	br.Watch("test", "data")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		br.Unwatch("test", "data")
		br.Watch("test", "data")
	}
}

func BenchmarkBroadcast_Broadcast(b *testing.B) {
	br := New[string]()
	handler := func(signal string, data string, metadata map[string]interface{}) error {
		return nil
	}
	br.Handle(handler)
	br.Watch("test", "data1")
	br.Watch("test", "data2")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		br.Broadcast("test", nil)
	}
}

func BenchmarkBroadcast_ConcurrentBroadcast(b *testing.B) {
	br := New[string]()
	handler := func(signal string, data string, metadata map[string]interface{}) error {
		return nil
	}
	br.Handle(handler)

	for i := 0; i < 100; i++ {
		br.Watch("test", "data")
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			br.Broadcast("test", nil)
		}
	})
}

func TestBroadcast_HandlerExecution(t *testing.T) {
	tests := []struct {
		name          string
		signals       []string
		data          []string
		handlerCount  int
		expectedCalls int
	}{
		{
			name:          "single handler single signal",
			signals:       []string{"test"},
			data:          []string{"data"},
			handlerCount:  1,
			expectedCalls: 1,
		},
		{
			name:          "multiple handlers single signal",
			signals:       []string{"test"},
			data:          []string{"data"},
			handlerCount:  3,
			expectedCalls: 3,
		},
		{
			name:          "single handler multiple signals",
			signals:       []string{"test1", "test2"},
			data:          []string{"data1", "data2"},
			handlerCount:  1,
			expectedCalls: 2,
		},
		{
			name:          "multiple handlers multiple signals",
			signals:       []string{"test1", "test2"},
			data:          []string{"data1", "data2"},
			handlerCount:  2,
			expectedCalls: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New[string]()
			calls := 0

			// Register handlers
			for i := 0; i < tt.handlerCount; i++ {
				b.Handle(func(signal string, data string, metadata map[string]interface{}) error {
					calls++
					return nil
				})
			}

			// Register watchers
			for i, signal := range tt.signals {
				b.Watch(signal, tt.data[i])
			}

			// Broadcast all signals
			for _, signal := range tt.signals {
				b.Broadcast(signal, nil)
			}

			if calls != tt.expectedCalls {
				t.Errorf("expected %d handler calls, got %d", tt.expectedCalls, calls)
			}
		})
	}
}

type TestData struct {
	ID      int
	Name    string
	Payload map[string]interface{}
}

func TestBroadcast_StructData(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*Broadcast[*TestData])
		validate func(*testing.T, *Broadcast[*TestData])
	}{
		{
			name: "handle struct data",
			setup: func(b *Broadcast[*TestData]) {
				data1 := &TestData{
					ID:   1,
					Name: "test1",
					Payload: map[string]interface{}{
						"key1": "value1",
					},
				}
				data2 := &TestData{
					ID:   1,
					Name: "test1",
					Payload: map[string]interface{}{
						"key1": "value1",
					},
				}
				b.Watch("test", data1)
				b.Watch("test", data2) // Should be considered duplicate
			},
			validate: func(t *testing.T, b *Broadcast[*TestData]) {
				if len(b.listeners["test"]) != 2 {
					t.Errorf("expected 2 listeners for struct data, got %d", len(b.listeners["test"]))
				}
			},
		},
		{
			name: "multiple different struct data",
			setup: func(b *Broadcast[*TestData]) {
				data1 := &TestData{
					ID:   1,
					Name: "test1",
					Payload: map[string]interface{}{
						"key1": "value1",
					},
				}
				data2 := &TestData{
					ID:   2,
					Name: "test2",
					Payload: map[string]interface{}{
						"key2": "value2",
					},
				}
				b.Watch("test", data1)
				b.Watch("test", data2)
			},
			validate: func(t *testing.T, b *Broadcast[*TestData]) {
				if len(b.listeners["test"]) != 2 {
					t.Errorf("expected 2 listeners for different struct data, got %d", len(b.listeners["test"]))
				}
			},
		},
		{
			name: "unwatch struct data",
			setup: func(b *Broadcast[*TestData]) {
				data := &TestData{
					ID:   1,
					Name: "test1",
					Payload: map[string]interface{}{
						"key1": "value1",
					},
				}
				b.Watch("test", data)
				b.Unwatch("test", data)
			},
			validate: func(t *testing.T, b *Broadcast[*TestData]) {
				if len(b.listeners["test"]) != 0 {
					t.Errorf("expected 0 listeners after unwatch, got %d", len(b.listeners["test"]))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New[*TestData]()
			tt.setup(b)
			tt.validate(t, b)
		})
	}
}

type TestDataUniquer struct {
	ID   int
	Name string
}

func TestBroadcast_StructDataUniquer(t *testing.T) {
	b := New[TestDataUniquer]()

	data1 := TestDataUniquer{ID: 1, Name: "test1"}
	data2 := TestDataUniquer{ID: 1, Name: "test1"}
	data3 := TestDataUniquer{ID: 2, Name: "test3"}

	calls := 0
	b.Handle(func(signal string, data TestDataUniquer, metadata map[string]interface{}) error {
		calls++
		return nil
	})

	b.Watch("test", data1)
	b.Watch("test", data2)
	b.Watch("test", data3)

	if len(b.listeners["test"]) != 2 {
		t.Errorf("expected 2 listeners, got %d", len(b.listeners["test"]))
	}

	b.Broadcast("test", nil)
	if calls != 2 {
		t.Errorf("expected 2 handler calls, got %d", calls)
	}
}

func TestBroadcast_StructDataHandling(t *testing.T) {
	b := New[TestDataUniquer]()
	receivedData := make([]TestDataUniquer, 0)
	mutex := sync.Mutex{}

	handler := func(signal string, data TestDataUniquer, metadata map[string]interface{}) error {
		mutex.Lock()
		receivedData = append(receivedData, data)
		mutex.Unlock()
		return nil
	}
	b.Handle(handler)

	testData := []TestDataUniquer{
		{
			ID:   1,
			Name: "test1",
		},
		{
			ID:   2,
			Name: "test2",
		},
	}

	// Watch both data
	for _, data := range testData {
		b.Watch("test", data)
	}

	// Broadcast and verify
	b.Broadcast("test", nil)

	if len(receivedData) != len(testData) {
		t.Errorf("expected %d received data, got %d", len(testData), len(receivedData))
	}

	// Verify data integrity
	for i, data := range testData {
		if receivedData[i].ID != data.ID ||
			receivedData[i].Name != data.Name {
			t.Errorf("data at index %d does not match original data", i)
		}
	}
}

func BenchmarkBroadcast_StructData(b *testing.B) {
	br := New[*TestData]()
	data := &TestData{
		ID:   1,
		Name: "test",
		Payload: map[string]interface{}{
			"key": "value",
		},
	}

	handler := func(signal string, data *TestData, metadata map[string]interface{}) error {
		return nil
	}
	br.Handle(handler)
	br.Watch("test", data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		br.Broadcast("test", nil)
	}
}

func TestBroadcast_Clean(t *testing.T) {
	b := New[TestDataUniquer]()

	// 添加多个监听器到不同的信号
	signals := []string{"test1", "test2", "test3"}
	for _, signal := range signals {
		for i := 0; i < 3; i++ {
			data := TestDataUniquer{ID: i, Name: fmt.Sprintf("test_%s_%d", signal, i)}
			b.Watch(signal, data)
		}
	}

	// 验证初始状态
	for _, signal := range signals {
		if count := b.WatchCount(signal); count != 3 {
			t.Errorf("expected 3 watchers for signal %s, got %d", signal, count)
		}
	}

	// 测试清除单个信号
	b.Clean("test1")
	if b.HasWatch("test1") {
		t.Error("watchers for test1 should be removed")
	}
	if !b.HasWatch("test2") {
		t.Error("watchers for test2 should remain")
	}

	// 测试 CleanAll
	b.CleanAll()
	for _, signal := range signals {
		if b.HasWatch(signal) {
			t.Errorf("signal %s should have no watchers after CleanAll", signal)
		}
	}
}

func TestBroadcast_HasWatch(t *testing.T) {
	b := New[TestDataUniquer]()

	// 测试空信号
	if b.HasWatch("test") {
		t.Error("empty signal should not have watchers")
	}

	// 添加监听器
	data := TestDataUniquer{ID: 1, Name: "test"}
	b.Watch("test", data)

	// 测试有监听器的信号
	if !b.HasWatch("test") {
		t.Error("signal should have watchers after Watch")
	}

	// 清除后测试
	b.Clean("test")
	if b.HasWatch("test") {
		t.Error("signal should not have watchers after Clean")
	}
}

func TestBroadcast_WatchCount(t *testing.T) {
	b := New[TestDataUniquer]()

	// 测试空信号
	if count := b.WatchCount("test"); count != 0 {
		t.Errorf("empty signal should have 0 watchers, got %d", count)
	}

	// 添加多个监听器
	for i := 0; i < 3; i++ {
		data := TestDataUniquer{ID: i, Name: fmt.Sprintf("test%d", i)}
		b.Watch("test", data)
	}

	// 验证数量
	if count := b.WatchCount("test"); count != 3 {
		t.Errorf("expected 3 watchers, got %d", count)
	}

	// 测试添加重复数据
	duplicate := TestDataUniquer{ID: 0, Name: "duplicate"}
	b.Watch("test", duplicate)
	if count := b.WatchCount("test"); count != 3 {
		t.Errorf("watcher count should not increase for duplicate data, got %d", count)
	}
}

// 添加性能测试
func BenchmarkBroadcast_Clean(b *testing.B) {
	br := New[TestDataUniquer]()
	data := TestDataUniquer{ID: 1, Name: "test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		br.Watch("test", data)
		br.Clean("test")
	}
}

func BenchmarkBroadcast_CleanAll(b *testing.B) {
	br := New[TestDataUniquer]()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		br.CleanAll()
	}
}

func TestBroadcast_Range(t *testing.T) {
	b := New[TestDataUniquer]()

	// 准备测试数据
	expectedSignals := map[string]int{
		"signal1": 2,
		"signal2": 3,
		"signal3": 1,
	}

	for signal, count := range expectedSignals {
		for i := 0; i < count; i++ {
			data := TestDataUniquer{ID: i, Name: fmt.Sprintf("test_%s_%d", signal, i)}
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
	stopSignal := "signal2"
	visitCount := 0
	b.Range(func(signal string, count int) bool {
		visitCount++
		return signal != stopSignal
	})

	if visitCount > len(expectedSignals) {
		t.Errorf("Range should stop after finding stop signal, visited %d signals", visitCount)
	}
}

func BenchmarkBroadcast_Range(b *testing.B) {
	br := New[TestDataUniquer]()

	// 准备测试数据
	for i := 0; i < 100; i++ {
		signal := fmt.Sprintf("signal%d", i)
		data := TestDataUniquer{ID: i, Name: "test"}
		br.Watch(signal, data)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		br.Range(func(signal string, count int) bool {
			return true
		})
	}
}
