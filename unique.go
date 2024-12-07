package broadcast

import (
	"sync"
	"unique"
)

// Uniquer 接口定义了可以获取唯一标识和值的类型
type Uniquer[K comparable, T any] interface {
	Unique() unique.Handle[K]
	Value() T
}

// UniqueHandler 定义了处理 Uniquer 数据的处理器函数类型
type UniqueHandler[K comparable, T any] func(signal string, data T) error

// UniqueBroadcast 实现了对 Uniquer 类型数据的广播功能
type UniqueBroadcast[K comparable, T any] struct {
	mu        sync.RWMutex
	handlers  []UniqueHandler[K, T]
	listeners map[string][]Uniquer[K, T]
}

// Handle 注册一个处理器
func (b *UniqueBroadcast[K, T]) Handle(handler UniqueHandler[K, T]) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.handlers == nil {
		b.handlers = make([]UniqueHandler[K, T], 0)
	}
	b.handlers = append(b.handlers, handler)
}

// Watch 监听一个信号
func (b *UniqueBroadcast[K, T]) Watch(signal string, data Uniquer[K, T]) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.listeners == nil {
		b.listeners = make(map[string][]Uniquer[K, T])
	}

	listeners := b.listeners[signal]
	handle := data.Unique()
	for _, listener := range listeners {
		if listener.Unique() == handle {
			return
		}
	}

	// 创建新的切片以避免共享底层数组
	newListeners := make([]Uniquer[K, T], len(listeners)+1)
	copy(newListeners, listeners)
	newListeners[len(listeners)] = data
	b.listeners[signal] = newListeners
}

// Unwatch 取消监听一个信号
func (b *UniqueBroadcast[K, T]) Unwatch(signal string, data Uniquer[K, T]) {
	b.mu.Lock()
	defer b.mu.Unlock()

	listeners := b.listeners[signal]
	if listeners == nil {
		return
	}

	handle := data.Unique()
	for i, item := range listeners {
		if item.Unique() == handle {
			// 创建新的切片以避免共享底层数组
			newListeners := make([]Uniquer[K, T], 0, len(listeners)-1)
			newListeners = append(newListeners, listeners[:i]...)
			newListeners = append(newListeners, listeners[i+1:]...)
			b.listeners[signal] = newListeners
			break
		}
	}
}

// Broadcast 广播一个信号
func (b *UniqueBroadcast[K, T]) Broadcast(signal string) {
	// 获取快照以减少锁持有时间
	b.mu.RLock()
	listeners := make([]Uniquer[K, T], len(b.listeners[signal]))
	copy(listeners, b.listeners[signal])
	handlers := make([]UniqueHandler[K, T], len(b.handlers))
	copy(handlers, b.handlers)
	b.mu.RUnlock()

	// 使用快照数据执行回调
	for _, handler := range handlers {
		for _, data := range listeners {
			// 创建数据副本以避免并发访问
			dataCopy := data.Value()
			_ = handler(signal, dataCopy)
		}
	}
}

// HasWatch 检查指定信号是否有监听器
func (b *UniqueBroadcast[K, T]) HasWatch(signal string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	listeners, exists := b.listeners[signal]
	return exists && len(listeners) > 0
}

// WatchCount 返回指定信号的监听器数量
func (b *UniqueBroadcast[K, T]) WatchCount(signal string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.listeners[signal])
}

// Clean 清除指定信号的所有监听器
func (b *UniqueBroadcast[K, T]) Clean(signal string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.listeners, signal)
}

// CleanAll 清除所有信号的监听器
func (b *UniqueBroadcast[K, T]) CleanAll() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.listeners = make(map[string][]Uniquer[K, T])
}

// Range 遍历所有信号及其监听器数量
// 如果 fn 返回 false，则停止遍历
func (b *UniqueBroadcast[K, T]) Range(fn func(signal string, count int) bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for signal, listeners := range b.listeners {
		if !fn(signal, len(listeners)) {
			break
		}
	}
}
