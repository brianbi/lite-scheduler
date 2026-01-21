package lite_scheduler

import (
	"context"
	"testing"
	"time"
)

// TestSchedulerCreation 测试调度器创建
func TestSchedulerCreation(t *testing.T) {
	config := DefaultConfig()
	config.MaxConcurrent = 5
	config.CheckInterval = 100 * time.Millisecond

	scheduler := New(config)

	if scheduler == nil {
		t.Fatal("Expected scheduler to be created, got nil")
	}

	if scheduler.config.MaxConcurrent != 5 {
		t.Errorf("Expected MaxConcurrent to be 5, got %d", scheduler.config.MaxConcurrent)
	}

	if scheduler.config.CheckInterval != 100*time.Millisecond {
		t.Errorf("Expected CheckInterval to be 100ms, got %v", scheduler.config.CheckInterval)
	}
}

// TestAddTask 测试添加任务
func TestAddTask(t *testing.T) {
	scheduler := New(DefaultConfig())

	task := NewTask("test-task", "Test Task", func(ctx context.Context) error {
		return nil
	}).WithInterval(1 * time.Second)

	err := scheduler.AddTask(task)
	if err != nil {
		t.Fatalf("Expected no error when adding task, got %v", err)
	}

	if len(scheduler.tasks) != 1 {
		t.Errorf("Expected 1 task in scheduler, got %d", len(scheduler.tasks))
	}

	if _, exists := scheduler.tasks["test-task"]; !exists {
		t.Error("Expected task to exist in scheduler")
	}
}

// TestAddTaskDuplicateID 测试添加重复ID的任务
func TestAddTaskDuplicateID(t *testing.T) {
	scheduler := New(DefaultConfig())

	task1 := NewTask("test-task", "Test Task 1", func(ctx context.Context) error {
		return nil
	}).WithInterval(1 * time.Second)

	task2 := NewTask("test-task", "Test Task 2", func(ctx context.Context) error {
		return nil
	}).WithInterval(1 * time.Second)

	err := scheduler.AddTask(task1)
	if err != nil {
		t.Fatalf("Expected no error when adding first task, got %v", err)
	}

	err = scheduler.AddTask(task2)
	if err == nil {
		t.Fatal("Expected error when adding duplicate task ID, got nil")
	}

	if len(scheduler.tasks) != 1 {
		t.Errorf("Expected 1 task in scheduler, got %d", len(scheduler.tasks))
	}
}

// TestRemoveTask 测试移除任务
func TestRemoveTask(t *testing.T) {
	scheduler := New(DefaultConfig())

	task := NewTask("test-task", "Test Task", func(ctx context.Context) error {
		return nil
	}).WithInterval(1 * time.Second)

	err := scheduler.AddTask(task)
	if err != nil {
		t.Fatalf("Expected no error when adding task, got %v", err)
	}

	err = scheduler.RemoveTask("test-task")
	if err != nil {
		t.Fatalf("Expected no error when removing task, got %v", err)
	}

	if len(scheduler.tasks) != 0 {
		t.Errorf("Expected 0 tasks in scheduler, got %d", len(scheduler.tasks))
	}

	if _, exists := scheduler.tasks["test-task"]; exists {
		t.Error("Expected task to not exist in scheduler after removal")
	}
}

// TestRemoveNonExistentTask 测试移除不存在的任务
func TestRemoveNonExistentTask(t *testing.T) {
	scheduler := New(DefaultConfig())

	err := scheduler.RemoveTask("non-existent-task")
	if err == nil {
		t.Fatal("Expected error when removing non-existent task, got nil")
	}
}

// TestGetTask 测试获取任务
func TestGetTask(t *testing.T) {
	scheduler := New(DefaultConfig())

	task := NewTask("test-task", "Test Task", func(ctx context.Context) error {
		return nil
	}).WithInterval(1 * time.Second)

	err := scheduler.AddTask(task)
	if err != nil {
		t.Fatalf("Expected no error when adding task, got %v", err)
	}

	retrievedTask, err := scheduler.GetTask("test-task")
	if err != nil {
		t.Fatalf("Expected no error when getting task, got %v", err)
	}

	if retrievedTask.ID != task.ID {
		t.Errorf("Expected task ID to be %s, got %s", task.ID, retrievedTask.ID)
	}

	if retrievedTask.Name != task.Name {
		t.Errorf("Expected task name to be %s, got %s", task.Name, retrievedTask.Name)
	}
}

// TestGetNonExistentTask 测试获取不存在的任务
func TestGetNonExistentTask(t *testing.T) {
	scheduler := New(DefaultConfig())

	_, err := scheduler.GetTask("non-existent-task")
	if err == nil {
		t.Fatal("Expected error when getting non-existent task, got nil")
	}
}

// TestListTasks 测试列出任务
func TestListTasks(t *testing.T) {
	scheduler := New(DefaultConfig())

	task1 := NewTask("test-task-1", "Test Task 1", func(ctx context.Context) error {
		return nil
	}).WithInterval(1 * time.Second)

	task2 := NewTask("test-task-2", "Test Task 2", func(ctx context.Context) error {
		return nil
	}).WithInterval(2 * time.Second)

	err := scheduler.AddTask(task1)
	if err != nil {
		t.Fatalf("Expected no error when adding task 1, got %v", err)
	}

	err = scheduler.AddTask(task2)
	if err != nil {
		t.Fatalf("Expected no error when adding task 2, got %v", err)
	}

	tasks := scheduler.ListTasks()

	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks in scheduler, got %d", len(tasks))
	}

	// 验证任务列表包含正确的任务
	taskIDs := make(map[string]bool)
	for _, task := range tasks {
		taskIDs[task.ID] = true
	}

	if !taskIDs["test-task-1"] {
		t.Error("Expected test-task-1 to be in the list")
	}

	if !taskIDs["test-task-2"] {
		t.Error("Expected test-task-2 to be in the list")
	}
}

// TestPauseResumeTask 测试暂停和恢复任务
func TestPauseResumeTask(t *testing.T) {
	scheduler := New(DefaultConfig())

	task := NewTask("test-task", "Test Task", func(ctx context.Context) error {
		return nil
	}).WithInterval(1 * time.Second)

	err := scheduler.AddTask(task)
	if err != nil {
		t.Fatalf("Expected no error when adding task, got %v", err)
	}

	// 暂停任务
	err = scheduler.PauseTask("test-task")
	if err != nil {
		t.Fatalf("Expected no error when pausing task, got %v", err)
	}

	// 检查任务状态
	taskInfo, err := scheduler.GetTask("test-task")
	if err != nil {
		t.Fatalf("Expected no error when getting task, got %v", err)
	}

	if taskInfo.GetStatus() != TaskStatusPaused {
		t.Errorf("Expected task status to be Paused, got %s", taskInfo.GetStatus())
	}

	// 恢复任务
	err = scheduler.ResumeTask("test-task")
	if err != nil {
		t.Fatalf("Expected no error when resuming task, got %v", err)
	}

	// 检查任务状态
	taskInfo, err = scheduler.GetTask("test-task")
	if err != nil {
		t.Fatalf("Expected no error when getting task, got %v", err)
	}

	if taskInfo.GetStatus() != TaskStatusPending {
		t.Errorf("Expected task status to be Pending, got %s", taskInfo.GetStatus())
	}
}

// TestRunNow 测试立即运行任务
func TestRunNow(t *testing.T) {
	scheduler := New(DefaultConfig())

	ran := make(chan bool, 1)

	task := NewTask("test-task", "Test Task", func(ctx context.Context) error {
		ran <- true
		return nil
	}).WithInterval(10 * time.Second) // 很长的间隔，确保不会自动触发

	err := scheduler.AddTask(task)
	if err != nil {
		t.Fatalf("Expected no error when adding task, got %v", err)
	}

	err = scheduler.RunNow("test-task")
	if err != nil {
		t.Fatalf("Expected no error when running task now, got %v", err)
	}

	// 等待任务执行
	select {
	case <-ran:
		// 任务成功执行
	case <-time.After(2 * time.Second):
		t.Error("Expected task to run within 2 seconds")
	}
}

// TestRunNowNonExistentTask 测试对不存在的任务执行RunNow
func TestRunNowNonExistentTask(t *testing.T) {
	scheduler := New(DefaultConfig())

	err := scheduler.RunNow("non-existent-task")
	if err == nil {
		t.Fatal("Expected error when running non-existent task, got nil")
	}
}

// TestSchedulerStartStop 测试调度器启动和停止
func TestSchedulerStartStop(t *testing.T) {
	scheduler := New(DefaultConfig())

	// 检查初始状态
	if scheduler.running {
		t.Error("Expected scheduler to not be running initially")
	}

	// 启动调度器
	scheduler.Start()

	// 等待一段时间确保启动完成
	time.Sleep(50 * time.Millisecond)

	// 检查运行状态
	if !scheduler.running {
		t.Error("Expected scheduler to be running after start")
	}

	// 停止调度器
	scheduler.Stop()

	// 检查停止状态
	if scheduler.running {
		t.Error("Expected scheduler to not be running after stop")
	}
}

// TestGetStatus 测试获取调度器状态
func TestGetStatus(t *testing.T) {
	scheduler := New(DefaultConfig())

	task := NewTask("test-task", "Test Task", func(ctx context.Context) error {
		return nil
	}).WithInterval(1 * time.Second)

	err := scheduler.AddTask(task)
	if err != nil {
		t.Fatalf("Expected no error when adding task, got %v", err)
	}

	status := scheduler.GetStatus()

	if status.TotalTasks != 1 {
		t.Errorf("Expected TotalTasks to be 1, got %d", status.TotalTasks)
	}

	if status.MaxConcurrent != DefaultConfig().MaxConcurrent {
		t.Errorf("Expected MaxConcurrent to be %d, got %d", DefaultConfig().MaxConcurrent, status.MaxConcurrent)
	}

	if status.CurrentTime.IsZero() {
		t.Error("Expected CurrentTime to not be zero")
	}
}

// TestSchedulerConfigurableConcurrency 测试可配置的并发数
func TestSchedulerConfigurableConcurrency(t *testing.T) {
	config := DefaultConfig()
	config.MaxConcurrent = 3

	scheduler := New(config)

	if scheduler.config.MaxConcurrent != 3 {
		t.Errorf("Expected MaxConcurrent to be 3, got %d", scheduler.config.MaxConcurrent)
	}

	// 检查信号量容量
	if cap(scheduler.semaphore) != 3 {
		t.Errorf("Expected semaphore capacity to be 3, got %d", cap(scheduler.semaphore))
	}
}

// TestSchedulerCheckInterval 测试检查间隔配置
func TestSchedulerCheckInterval(t *testing.T) {
	config := DefaultConfig()
	config.CheckInterval = 200 * time.Millisecond

	scheduler := New(config)

	if scheduler.config.CheckInterval != 200*time.Millisecond {
		t.Errorf("Expected CheckInterval to be 200ms, got %v", scheduler.config.CheckInterval)
	}
}

// TestSchedulerEnableRecovery 测试故障恢复配置
func TestSchedulerEnableRecovery(t *testing.T) {
	config := DefaultConfig()
	config.EnableRecovery = false

	scheduler := New(config)

	// 验证配置被正确应用
	if scheduler.config.EnableRecovery != false {
		t.Errorf("Expected EnableRecovery to be false, got %v", scheduler.config.EnableRecovery)
	}
}
