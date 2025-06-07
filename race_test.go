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

// TestData for concurrent testing
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

// TestRaceBroadcast_ConcurrentOperations tests thread safety of all operations
func TestRaceBroadcast_ConcurrentOperations(t *testing.T) {
	b := &UniqueBroadcast[int, concurrentTestData]{}
	var wg sync.WaitGroup
	const (
		numGoroutines = 10
		numOperations = 1000
	)

	// Add multiple handlers
	handlerCounter := uint64(0)
	for i := 0; i < 5; i++ {
		b.Handle(func(signal string, data concurrentTestData, metadata map[string]interface{}) error {
			atomic.AddUint64(&handlerCounter, 1)
			return nil
		})
	}

	// Concurrent execution of Watch, Unwatch and Broadcast operations
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
				time.Sleep(time.Microsecond) // Simulate real-world delay
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
				b.Broadcast("test", nil)
				time.Sleep(time.Microsecond)
			}
		}()
	}

	wg.Wait()
}

// TestRaceBroadcast_MultipleSignals tests concurrent operations with multiple signals
func TestRaceBroadcast_MultipleSignals(t *testing.T) {
	b := &UniqueBroadcast[int, concurrentTestData]{}
	var wg sync.WaitGroup
	const numSignals = 10
	const numOperationsPerSignal = 100

	signals := make([]string, numSignals)
	for i := 0; i < numSignals; i++ {
		signals[i] = fmt.Sprintf("signal-%d", i)
	}

	// Add handler
	handlerCalls := make(map[string]uint64)
	handlerMutex := sync.RWMutex{}

	b.Handle(func(signal string, data concurrentTestData, metadata map[string]interface{}) error {
		handlerMutex.Lock()
		handlerCalls[signal]++
		handlerMutex.Unlock()
		return nil
	})

	// Start concurrent operations for each signal
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
				b.Broadcast(sig, nil)
				time.Sleep(time.Microsecond)
			}
		}(signal)
	}

	wg.Wait()
}

// TestRaceBroadcast_HandlerModification tests concurrent handler modifications
func TestRaceBroadcast_HandlerModification(t *testing.T) {
	b := &UniqueBroadcast[int, concurrentTestData]{}
	var wg sync.WaitGroup
	const numOperations = 1000

	// Concurrent handler addition and triggering
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < numOperations; i++ {
			b.Handle(func(signal string, data concurrentTestData, metadata map[string]interface{}) error {
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
			b.Broadcast("test", nil)
		}
	}()

	wg.Wait()
}

// TestRaceBroadcast_ListenerModification tests concurrent listener modifications
func TestRaceBroadcast_ListenerModification(t *testing.T) {
	b := &UniqueBroadcast[int, concurrentTestData]{}
	var wg sync.WaitGroup
	const numOperations = 1000

	handlerCalled := uint64(0)
	b.Handle(func(signal string, data concurrentTestData, metadata map[string]interface{}) error {
		atomic.AddUint64(&handlerCalled, 1)
		return nil
	})

	// Concurrent modification of listeners and broadcasting
	wg.Add(3)

	// Add listeners
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

	// Remove listeners
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

	// Broadcast
	go func() {
		defer wg.Done()
		for i := 0; i < numOperations; i++ {
			b.Broadcast("test", nil)
			time.Sleep(time.Microsecond)
		}
	}()

	wg.Wait()
}
