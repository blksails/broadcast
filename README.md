# Broadcast

一个轻量级、类型安全的事件广播系统，支持泛型和并发安全。

## 特性

- 🚀 高性能：优化的并发处理和内存使用
- 🔒 并发安全：内置线程安全机制
- 📦 泛型支持：灵活的类型系统
- 🎯 唯一性保证：自动去重监听器
- 🔌 简单易用：直观的 API 设计

## 安装
bash
go get pkg.blksails.net/pkg/broadcast


## 基础用法
```go
// 创建广播实例
b := &broadcast.Broadcast[string]{}
// 注册处理器
b.Handle(func(signal string, data string) error {
	fmt.Printf("Received signal: %s, data: %s\n", signal, data)
	return nil
})
// 监听信号
b.Watch("user.login", "user123")
// 广播信号
b.Broadcast("user.login")
```

## 高级用法：Unique 广播

对于需要唯一性保证的复杂数据类型，可以使用 UniqueBroadcast：

```go
// 定义数据结构
type UserEvent struct {
    UserID int
    Action string
    Metadata map[string]string
}
// 实现 Uniquer 接口
type UserEventWrapper struct {
	event UserEvent
}

func (u UserEventWrapper) Unique() unique.Handle[int] {
	return unique.Make(u.event.UserID)
}
func (u UserEventWrapper) Value() UserEvent {
	return u.event
}
```
## 示例

完整的示例代码可以在 demo/main.go 中找到：

```bash
go run demo/main.go
```


## 性能基准测试

```bash
go test -bench=. -benchmem
```

基准测试结果示例：
- Watch 操作: ~500ns/op
- Broadcast 操作: ~1000ns/op
- 并发广播: ~2000ns/op

## 线程安全性

该库提供了完整的并发安全保证：
- 所有公共方法都是线程安全的
- 使用读写锁优化并发性能
- 支持高并发场景下的数据一致性

## API 文档

### Broadcast[T comparable]

基础广播类型，适用于简单数据类型：

- `Handle(handler Handler[T])`：注册信号处理器
- `Watch(signal string, data T)`：监听信号
- `Unwatch(signal string, data T)`：取消监听
- `Broadcast(signal string)`：广播信号

### UniqueBroadcast[K comparable, T any]

支持唯一性的广播类型，适用于复杂数据类型：

- `Handle(handler UniqueHandler[K, T])`：注册信号处理器
- `Watch(signal string, data Uniquer[K, T])`：监听信号
- `Unwatch(signal string, data Uniquer[K, T])`：取消监听
- `Broadcast(signal string)`：广播信号

## 贡献

欢迎提交 Pull Request 和 Issue！

## 许可证

MIT License