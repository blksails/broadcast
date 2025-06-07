package main

import (
	"fmt"
	"sync"
	"time"
	"unique"

	"pkg.blksails.net/x/broadcast"
)

// UserEvent 表示用户事件数据
type UserEvent struct {
	UserID   int
	Action   string
	Metadata map[string]string
}

// UserEventWrapper 包装 UserEvent 以实现 Uniquer 接口
type UserEventWrapper struct {
	event UserEvent
}

func (u *UserEventWrapper) Unique() unique.Handle[int] {
	return unique.Make(u.event.UserID)
}

func (u *UserEventWrapper) Value() UserEvent {
	return u.event
}

func main() {
	// 创建广播实例
	b := &broadcast.UniqueBroadcast[int, UserEvent]{}

	// 添加事件处理器
	b.Handle(func(signal string, event UserEvent, metadata map[string]interface{}) error {
		fmt.Printf("[Handler 1] Signal: %s, UserID: %d, Action: %s\n",
			signal, event.UserID, event.Action)
		return nil
	})

	b.Handle(func(signal string, event UserEvent, metadata map[string]interface{}) error {
		fmt.Printf("[Handler 2] Signal: %s, UserID: %d, Action: %s\n",
			signal, event.UserID, event.Action)
		return nil
	})

	// 创建一些测试事件
	events := []*UserEventWrapper{
		{
			event: UserEvent{
				UserID: 1,
				Action: "login",
				Metadata: map[string]string{
					"ip": "192.168.1.1",
				},
			},
		},
		{
			event: UserEvent{
				UserID: 2,
				Action: "purchase",
				Metadata: map[string]string{
					"product": "item1",
				},
			},
		},
		{
			event: UserEvent{
				UserID: 1, // 相同的 UserID
				Action: "logout",
				Metadata: map[string]string{
					"ip": "192.168.1.1",
				},
			},
		},
	}

	// 演示基本功能
	fmt.Println("\n=== 基本功能演示 ===")
	for _, e := range events {
		b.Watch("user_activity", e)
	}

	fmt.Println("\n广播 user_activity 信号:")
	b.Broadcast("user_activity", nil)

	// 取消监听某个事件
	fmt.Println("\n取消监听 UserID=1 的事件:")
	b.Unwatch("user_activity", events[0])

	fmt.Println("\n再次广播 user_activity 信号:")
	b.Broadcast("user_activity", nil)

	// 演示并发处理
	fmt.Println("\n=== 并发处理演示 ===")
	var wg sync.WaitGroup

	// 并发添加监听器
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			event := &UserEventWrapper{
				event: UserEvent{
					UserID:   id + 100,
					Action:   fmt.Sprintf("action_%d", id),
					Metadata: map[string]string{"concurrent": "true"},
				},
			}
			b.Watch("concurrent_activity", event)
			time.Sleep(time.Millisecond * 100) // 模拟实际工作负载
		}(i)
	}

	// 并发广播
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			time.Sleep(time.Millisecond * 50) // 给一些时间让监听器注册
			fmt.Printf("\n并发广播 #%d:\n", i+1)
			b.Broadcast("concurrent_activity", nil)
		}(i)
	}

	wg.Wait()

	// 演示不同信号
	fmt.Println("\n=== 多信号演示 ===")
	signals := []string{"login", "logout", "purchase"}
	for i, signal := range signals {
		event := &UserEventWrapper{
			event: UserEvent{
				UserID:   1000 + i,
				Action:   fmt.Sprintf("action_for_%s", signal),
				Metadata: map[string]string{"signal": signal},
			},
		}
		b.Watch(signal, event)
	}

	for _, signal := range signals {
		fmt.Printf("\n广播 %s 信号:\n", signal)
		b.Broadcast(signal, nil)
	}

	// ... 在现有代码后添加 ...

	fmt.Println("\n=== Watch 状态检查演示 ===")

	checkSignal := "status_check"
	fmt.Printf("\n1. 空信号状态:\n")
	fmt.Printf("信号 '%s' 是否有监听: %v\n", checkSignal, b.HasWatch(checkSignal))
	fmt.Printf("信号 '%s' 的监听数量: %d\n", checkSignal, b.WatchCount(checkSignal))

	fmt.Printf("\n2. 添加监听器:\n")
	for i := 0; i < 3; i++ {
		event := &UserEventWrapper{
			event: UserEvent{
				UserID:   4000 + i,
				Action:   fmt.Sprintf("status_action_%d", i),
				Metadata: map[string]string{"status": "active"},
			},
		}
		b.Watch(checkSignal, event)
		fmt.Printf("添加监听器 #%d 后的状态:\n", i+1)
		fmt.Printf("- 是否有监听: %v\n", b.HasWatch(checkSignal))
		fmt.Printf("- 监听数量: %d\n", b.WatchCount(checkSignal))
	}

	fmt.Printf("\n3. 添加重复ID的监听器:\n")
	duplicateEvent := &UserEventWrapper{
		event: UserEvent{
			UserID:   4000, // 重复的 ID
			Action:   "duplicate_action",
			Metadata: map[string]string{"duplicate": "true"},
		},
	}
	b.Watch(checkSignal, duplicateEvent)
	fmt.Printf("添加重复ID后的监听数量: %d\n", b.WatchCount(checkSignal))

	fmt.Printf("\n4. 清除信号:\n")
	b.Clean(checkSignal)
	fmt.Printf("清除后是否有监听: %v\n", b.HasWatch(checkSignal))
	fmt.Printf("清除后的监听数量: %d\n", b.WatchCount(checkSignal))

	fmt.Println("\n=== Range 遍历演示 ===")

	// 准备一些测试数据
	signals = []string{"login", "logout", "purchase", "view"}
	for i, signal := range signals {
		for j := 0; j < i+1; j++ { // 每个信号有不同数量的监听器
			event := &UserEventWrapper{
				event: UserEvent{
					UserID:   5000 + i*100 + j,
					Action:   fmt.Sprintf("action_%s_%d", signal, j),
					Metadata: map[string]string{"signal": signal},
				},
			}
			b.Watch(signal, event)
		}
	}

	fmt.Println("\n1. 遍历所有信号:")
	b.Range(func(signal string, count int) bool {
		fmt.Printf("信号: %-10s 监听器数量: %d\n", signal, count)
		return true
	})

	fmt.Println("\n2. 条件遍历 (只显示监听器数量大于1的信号):")
	b.Range(func(signal string, count int) bool {
		if count > 1 {
			fmt.Printf("信号: %-10s 监听器数量: %d\n", signal, count)
		}
		return true
	})

	fmt.Println("\n3. 提前终止遍历 (遇到 'logout' 信号停止):")
	b.Range(func(signal string, count int) bool {
		fmt.Printf("信号: %-10s 监听器数量: %d\n", signal, count)
		return signal != "logout"
	})
}
