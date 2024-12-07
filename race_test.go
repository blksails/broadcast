package broadcast

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unique"
)

// 并发测试辅助结构体
type concurrentTestData struct {
	ID    int
	Value string
}

type concurrentUniquer struct {
	data concurrentTestData
}

func (c *concurrentUniquer) Unique() unique.Handle[int] {
	return unique.Make(c.data.ID)
}

func (c *concurrentUniquer) Value() concurrentTestData {
	return c.data
}

// TestRaceBroadcast_ConcurrentOperations 测试所有操作的并发安全性
func TestRaceBroadcast_ConcurrentOperations(t *testing.T) {
	b := &UniqueBroadcast[int, concurrentTestData]{}
	var wg sync.WaitGroup
	const (
		numGoroutines = 10
		numOperations = 1000
	)

	// 添加多个处理器
	handlerCounter := uint64(0)
	for i := 0; i < 5; i++ {
		b.Handle(func(signal string, data concurrentTestData) error {
			atomic.AddUint64(&handlerCounter, 1)
			return nil
		})
	}

	// 并发执行 Watch、Unwatch 和 Broadcast 操作
	for i := 0; i < numGoroutines; i++ {
		wg.Add(3)

		// Watch goroutine
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				data := &concurrentUniquer{
					data: concurrentTestData{
						ID:    rand.Intn(100),
						Value: fmt.Sprintf("value-%d-%d", routineID, j),
					},
				}
				b.Watch("test", data)
				time.Sleep(time.Microsecond) // 模拟真实场景的间隔
			}
		}(i)

		// Unwatch goroutine
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				data := &concurrentUniquer{
					data: concurrentTestData{
						ID:    rand.Intn(100),
						Value: fmt.Sprintf("value-%d-%d", routineID, j),
					},
				}
				b.Unwatch("test", data)
				time.Sleep(time.Microsecond)
			}
		}(i)

		// Broadcast goroutine
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				b.Broadcast("test")
				time.Sleep(time.Microsecond)
			}
		}()
	}

	wg.Wait()
}

// TestRaceBroadcast_MultipleSignals 测试多信号并发
func TestRaceBroadcast_MultipleSignals(t *testing.T) {
	b := &UniqueBroadcast[int, concurrentTestData]{}
	var wg sync.WaitGroup
	const numSignals = 10
	const numOperationsPerSignal = 100

	signals := make([]string, numSignals)
	for i := 0; i < numSignals; i++ {
		signals[i] = fmt.Sprintf("signal-%d", i)
	}

	// 添加处理器
	handlerCalls := make(map[string]uint64)
	handlerMutex := sync.RWMutex{}

	b.Handle(func(signal string, data concurrentTestData) error {
		handlerMutex.Lock()
		handlerCalls[signal]++
		handlerMutex.Unlock()
		return nil
	})

	// 为每个信号启动并发操作
	for _, signal := range signals {
		wg.Add(3)

		// Watch
		go func(sig string) {
			defer wg.Done()
			for i := 0; i < numOperationsPerSignal; i++ {
				data := &concurrentUniquer{
					data: concurrentTestData{
						ID:    rand.Intn(100),
						Value: fmt.Sprintf("value-%s-%d", sig, i),
					},
				}
				b.Watch(sig, data)
			}
		}(signal)

		// Unwatch
		go func(sig string) {
			defer wg.Done()
			for i := 0; i < numOperationsPerSignal; i++ {
				data := &concurrentUniquer{
					data: concurrentTestData{
						ID:    rand.Intn(100),
						Value: fmt.Sprintf("value-%s-%d", sig, i),
					},
				}
				b.Unwatch(sig, data)
			}
		}(signal)

		// Broadcast
		go func(sig string) {
			defer wg.Done()
			for i := 0; i < numOperationsPerSignal; i++ {
				b.Broadcast(sig)
				time.Sleep(time.Microsecond)
			}
		}(signal)
	}

	wg.Wait()
}

// TestRaceBroadcast_HandlerModification 测试处理器并发修改
func TestRaceBroadcast_HandlerModification(t *testing.T) {
	b := &UniqueBroadcast[int, concurrentTestData]{}
	var wg sync.WaitGroup
	const numOperations = 1000

	// 并发添加和触发处理器
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < numOperations; i++ {
			b.Handle(func(signal string, data concurrentTestData) error {
				return nil
			})
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < numOperations; i++ {
			data := &concurrentUniquer{
				data: concurrentTestData{
					ID:    i,
					Value: fmt.Sprintf("value-%d", i),
				},
			}
			b.Watch("test", data)
			b.Broadcast("test")
		}
	}()

	wg.Wait()
}

// TestRaceBroadcast_ListenerModification 测试监听器并发修改
func TestRaceBroadcast_ListenerModification(t *testing.T) {
	b := &UniqueBroadcast[int, concurrentTestData]{}
	var wg sync.WaitGroup
	const numOperations = 1000

	handlerCalled := uint64(0)
	b.Handle(func(signal string, data concurrentTestData) error {
		atomic.AddUint64(&handlerCalled, 1)
		return nil
	})

	// 并发修改监听器和广播
	wg.Add(3)

	// 添加监听器
	go func() {
		defer wg.Done()
		for i := 0; i < numOperations; i++ {
			data := &concurrentUniquer{
				data: concurrentTestData{
					ID:    i,
					Value: fmt.Sprintf("value-%d", i),
				},
			}
			b.Watch("test", data)
		}
	}()

	// 移除监听器
	go func() {
		defer wg.Done()
		for i := 0; i < numOperations; i++ {
			data := &concurrentUniquer{
				data: concurrentTestData{
					ID:    i,
					Value: fmt.Sprintf("value-%d", i),
				},
			}
			b.Unwatch("test", data)
		}
	}()

	// ���播
	go func() {
		defer wg.Done()
		for i := 0; i < numOperations; i++ {
			b.Broadcast("test")
			time.Sleep(time.Microsecond)
		}
	}()

	wg.Wait()
}
