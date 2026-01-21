package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	scheduler "github.com/brianbi/lite-scheduler"
)

func main() {
	// 创建调度器
	config := scheduler.SchedulerConfig{
		MaxConcurrent:  5,
		CheckInterval:  time.Second,
		EnableRecovery: true,
	}
	sched := scheduler.New(config)

	// 设置日志
	logger := log.New(os.Stdout, "[Scheduler] ", log.LstdFlags|log.Lshortfile)
	sched.SetLogger(logger)

	// 注册事件处理器
	sched.OnEvent(func(event scheduler.Event) {
		switch event.Type {
		case scheduler.EventTaskStarted:
			fmt.Printf("📌 Task started: %s\n", event.Task.Name)
		case scheduler.EventTaskCompleted:
			fmt.Printf("✅ Task completed: %s\n", event.Task.Name)
		case scheduler.EventTaskFailed:
			fmt.Printf("❌ Task failed: %s - %v\n", event.Task.Name, event.Error)
		}
	})

	// 添加示例任务

	// 1. 每5秒执行的任务
	task1 := scheduler.NewTask("task-1", "心跳检测", func(ctx context.Context) error {
		fmt.Println("  → 执行心跳检测...")
		time.Sleep(time.Second)
		return nil
	}).WithInterval(5 * time.Second).WithTimeout(10 * time.Second)

	// 2. 使用Cron表达式的任务（每分钟的第0和30秒执行）
	task2 := scheduler.NewTask("task-2", "数据同步", func(ctx context.Context) error {
		fmt.Println("  → 执行数据同步...")
		time.Sleep(2 * time.Second)
		return nil
	}).WithCron("0,30 * * * * *").WithTimeout(30 * time.Second)

	// 3. 可能失败的任务（带重试）
	task3 := scheduler.NewTask("task-3", "外部API调用", func(ctx context.Context) error {
		fmt.Println("  → 调用外部API...")
		if rand.Float32() < 0.5 {
			return fmt.Errorf("API调用失败")
		}
		return nil
	}).WithInterval(10*time.Second).WithRetry(3, 2*time.Second)

	// 4. 长时间运行的任务
	task4 := scheduler.NewTask("task-4", "数据分析", func(ctx context.Context) error {
		fmt.Println("  → 开始数据分析...")
		select {
		case <-time.After(5 * time.Second):
			fmt.Println("  → 数据分析完成")
			return nil
		case <-ctx.Done():
			fmt.Println("  → 数据分析被取消")
			return ctx.Err()
		}
	}).WithInterval(20 * time.Second).WithTimeout(30 * time.Second)

	// 添加任务到调度器
	if err := sched.AddTask(task1); err != nil {
		log.Fatalf("Failed to add task1: %v", err)
	}
	if err := sched.AddTask(task2); err != nil {
		log.Fatalf("Failed to add task2: %v", err)
	}
	if err := sched.AddTask(task3); err != nil {
		log.Fatalf("Failed to add task3: %v", err)
	}
	if err := sched.AddTask(task4); err != nil {
		log.Fatalf("Failed to add task4: %v", err)
	}

	// 启动调度器
	sched.Start()
	fmt.Println("Scheduler started. Press Ctrl+C to stop.")

	// 显示任务状态的goroutine
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			fmt.Println("\n========== 任务状态 ==========")
			for _, info := range sched.ListTasks() {
				fmt.Printf("  %s: %s (运行: %d, 失败: %d, 下次: %s)\n",
					info.Name, info.Status, info.RunCount, info.FailCount,
					info.NextRunTime.Format("15:04:05"))
			}
			status := sched.GetStatus()
			fmt.Printf("  调度器: 运行中=%v, 当前并发=%d\n",
				status.Running, status.RunningTasks)
			fmt.Println("================================\n")
		}
	}()

	// 演示暂停和恢复
	go func() {
		time.Sleep(15 * time.Second)
		fmt.Println("\n⏸️  暂停任务: 心跳检测")
		sched.PauseTask("task-1")

		time.Sleep(10 * time.Second)
		fmt.Println("\n▶️  恢复任务: 心跳检测")
		sched.ResumeTask("task-1")
	}()

	// 等待中断信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down...")
	sched.Stop()
	fmt.Println("Scheduler stopped.")
}
