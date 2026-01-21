# Lite-Scheduler

Lite-Scheduler 是一个轻量级的 Go 语言任务调度库，支持基于 Cron 表达式和固定间隔的任务调度。

## 特性

- 🚀 轻量级设计，易于集成
- ⏰ 支持 Cron 表达式和固定间隔调度
- 🔄 自动重试机制
- 📊 任务状态管理
- 🛡️ 并发控制
- 🔔 事件通知系统
- 📝 详细日志记录
- 🛠️ 可配置的调度参数

## 安装

```bash
go get github.com/brianbi/lite-scheduler
```

## 快速开始

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	lite_scheduler "github.com/brianbi/lite-scheduler"
)

func main() {
	// 创建调度器
	config := lite_scheduler.DefaultConfig()
	config.MaxConcurrent = 5
	config.CheckInterval = time.Second
	
	scheduler := lite_scheduler.New(config)
	
	// 创建一个每5秒执行的任务
	task := lite_scheduler.NewTask("my-task", "My First Task", func(ctx context.Context) error {
		fmt.Println("任务执行中...")
		return nil
	}).WithInterval(5 * time.Second)
	
	// 添加任务到调度器
	err := scheduler.AddTask(task)
	if err != nil {
		log.Fatal("添加任务失败:", err)
	}
	
	// 启动调度器
	scheduler.Start()
	
	// 运行一段时间后停止
	time.Sleep(30 * time.Second)
	scheduler.Stop()
}
```

## 功能详解

### 1. 任务创建

```go
// 创建基本任务
task := lite_scheduler.NewTask("task-id", "Task Name", myFunction)

// 设置 Cron 表达式 (秒 分 时 日 月 周)
task.WithCron("0 */5 * * * *") // 每5分钟执行

// 设置固定间隔
task.WithInterval(10 * time.Second) // 每10秒执行

// 设置超时时间
task.WithTimeout(30 * time.Second)

// 设置重试策略
task.WithRetry(3, 5*time.Second) // 最多重试3次，每次间隔5秒
```

### 2. 任务管理

```go
// 添加任务
scheduler.AddTask(task)

// 获取任务
task, err := scheduler.GetTask("task-id")

// 删除任务
scheduler.RemoveTask("task-id")

// 暂停任务
scheduler.PauseTask("task-id")

// 恢复任务
scheduler.ResumeTask("task-id")

// 立即执行任务
scheduler.RunNow("task-id")
```

### 3. 事件监听

```go
// 注册事件处理器
scheduler.OnEvent(func(event lite_scheduler.Event) {
	switch event.Type {
	case lite_scheduler.EventTaskStarted:
		fmt.Printf("任务开始: %s\n", event.Task.Name)
	case lite_scheduler.EventTaskCompleted:
		fmt.Printf("任务完成: %s\n", event.Task.Name)
	case lite_scheduler.EventTaskFailed:
		fmt.Printf("任务失败: %s - %v\n", event.Task.Name, event.Error)
	}
})
```

### 4. 任务状态查询

```go
// 获取任务列表
tasks := scheduler.ListTasks()

// 获取调度器状态
status := scheduler.GetStatus()
fmt.Printf("运行中任务数: %d, 总任务数: %d\n", status.RunningTasks, status.TotalTasks)
```

## 任务状态

- `TaskStatusPending`: 等待中
- `TaskStatusRunning`: 运行中
- `TaskStatusCompleted`: 已完成
- `TaskStatusFailed`: 失败
- `TaskStatusCancelled`: 已取消
- `TaskStatusPaused`: 已暂停

## 配置选项

- `MaxConcurrent`: 最大并发任务数
- `CheckInterval`: 检查间隔
- `EnableRecovery`: 是否启用故障恢复

## Cron 表达式格式

支持 6 位 Cron 表达式格式: 秒 分 时 日 月 周

示例：
- `"0 */5 * * * *"` - 每5分钟执行
- `"0 0 */1 * * *"` - 每小时执行
- `"0 0 0 * * *"` - 每天零点执行
- `"0,30 * * * * *"` - 每分钟的第0秒和第30秒执行

## 错误处理

任务函数返回的错误会被捕获并记录，调度器会根据配置的重试策略进行重试。

## 许可证

MIT License