package broadcast

import (
	"sync"
	"unique"
)

type Handler[T comparable] func(signal string, data T) error

type Broadcast[T comparable] struct {
	mu        sync.RWMutex
	handlers  []Handler[T]
	listeners map[string][]unique.Handle[T]
}

// Handle 注册一个处理器
func (b *Broadcast[T]) Handle(handler Handler[T]) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.handlers == nil {
		b.handlers = make([]Handler[T], 0)
	}
	b.handlers = append(b.handlers, handler)
}

type uniqueWrapper[T comparable] struct {
	data T
}

func (u *uniqueWrapper[T]) Unique() unique.Handle[T] {
	return unique.Make(u.data)
}

func (u *uniqueWrapper[T]) Value() T {
	return u.data
}

// Watch 监听一个信号
func (b *Broadcast[T]) Watch(signal string, data T) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.listeners == nil {
		b.listeners = make(map[string][]unique.Handle[T])
	}

	var (
		handle    = unique.Make(data)
		listeners = b.listeners[signal]
	)
	for _, listener := range listeners {
		if listener == handle {
			return
		}
	}

	b.listeners[signal] = append(b.listeners[signal], handle)
}

// Unwatch 取消监听一个信号
func (b *Broadcast[T]) Unwatch(signal string, data T) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var (
		handle    = unique.Make(data)
		listeners = b.listeners[signal]
	)
	if listeners == nil {
		return
	}

	for i, item := range listeners {
		if item == handle {
			b.listeners[signal] = append(listeners[:i], listeners[i+1:]...)
			break
		}
	}
}

// Broadcast 广播一个信号, 以触发所有监听该信号的处理器
func (b *Broadcast[T]) Broadcast(signal string) {
	b.mu.RLock()
	listeners := b.listeners[signal]
	handlers := b.handlers
	b.mu.RUnlock()

	for _, handler := range handlers {
		for _, data := range listeners {
			_ = handler(signal, data.Value())
		}
	}
}

// Clean 清除指定信号的所有监听器
func (b *Broadcast[T]) Clean(signal string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.listeners, signal)
}

// CleanAll 清除所有信号的监听器
func (b *Broadcast[T]) CleanAll() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.listeners = make(map[string][]unique.Handle[T])
}

// HasWatch 检查指定信号是否有监听器
func (b *Broadcast[T]) HasWatch(signal string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	listeners, exists := b.listeners[signal]
	return exists && len(listeners) > 0
}

// WatchCount 返回指定信号的监听器数量
func (b *Broadcast[T]) WatchCount(signal string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.listeners[signal])
}

// Range 遍历所有信号及其监听器数量
// 如果 fn 返回 false，则停止遍历
func (b *Broadcast[T]) Range(fn func(signal string, count int) bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for signal, listeners := range b.listeners {
		if !fn(signal, len(listeners)) {
			break
		}
	}
}

// New 创建一个新的广播实例
func New[T comparable]() *Broadcast[T] {
	return &Broadcast[T]{
		handlers:  make([]Handler[T], 0),
		listeners: make(map[string][]unique.Handle[T]),
	}
}

// NewUnique 创建一个新的 UniqueBroadcast 实例
func NewUnique[K comparable, T any]() *UniqueBroadcast[K, T] {
	return &UniqueBroadcast[K, T]{
		handlers:  make([]UniqueHandler[K, T], 0),
		listeners: make(map[string][]Uniquer[K, T]),
	}
}
