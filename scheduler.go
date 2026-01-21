package lite_scheduler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// EventType 事件类型
type EventType int

const (
	EventTaskAdded EventType = iota
	EventTaskRemoved
	EventTaskStarted
	EventTaskCompleted
	EventTaskFailed
	EventTaskPaused
	EventTaskResumed
)

// Event 调度事件
type Event struct {
	Type      EventType
	Task      *Task
	Timestamp time.Time
	Error     error
}

// EventHandler 事件处理器
type EventHandler func(event Event)

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	MaxConcurrent  int           // 最大并发任务数
	CheckInterval  time.Duration // 检查间隔
	EnableRecovery bool          // 启用故障恢复
}

// DefaultConfig 默认配置
func DefaultConfig() SchedulerConfig {
	return SchedulerConfig{
		MaxConcurrent:  10,
		CheckInterval:  time.Second,
		EnableRecovery: true,
	}
}

// Scheduler 任务调度器
type Scheduler struct {
	config   SchedulerConfig
	tasks    map[string]*Task
	cronExpr map[string]*CronExpr
	running  bool

	// 并发控制
	semaphore chan struct{}
	wg        sync.WaitGroup
	mu        sync.RWMutex

	// 上下文管理
	ctx    context.Context
	cancel context.CancelFunc

	// 事件处理
	eventHandlers []EventHandler
	eventCh       chan Event

	// 日志
	logger *log.Logger
}

// New 创建新的调度器
func New(config SchedulerConfig) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	s := &Scheduler{
		config:    config,
		tasks:     make(map[string]*Task),
		cronExpr:  make(map[string]*CronExpr),
		semaphore: make(chan struct{}, config.MaxConcurrent),
		ctx:       ctx,
		cancel:    cancel,
		eventCh:   make(chan Event, 100),
		logger:    log.Default(),
	}

	// 启动事件处理协程
	go s.processEvents()

	return s
}

// SetLogger 设置日志
func (s *Scheduler) SetLogger(logger *log.Logger) {
	s.logger = logger
}

// OnEvent 注册事件处理器
func (s *Scheduler) OnEvent(handler EventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eventHandlers = append(s.eventHandlers, handler)
}

// processEvents 处理事件
func (s *Scheduler) processEvents() {
	for event := range s.eventCh {
		s.mu.RLock()
		handlers := s.eventHandlers
		s.mu.RUnlock()

		for _, handler := range handlers {
			handler(event)
		}
	}
}

// emitEvent 发送事件
func (s *Scheduler) emitEvent(eventType EventType, task *Task, err error) {
	select {
	case s.eventCh <- Event{
		Type:      eventType,
		Task:      task,
		Timestamp: time.Now(),
		Error:     err,
	}:
	default:
		s.logger.Println("Event channel full, dropping event")
	}
}

// AddTask 添加任务
func (s *Scheduler) AddTask(task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[task.ID]; exists {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}

	// 解析Cron表达式
	if task.CronExpr != "" {
		expr, err := ParseCron(task.CronExpr)
		if err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
		s.cronExpr[task.ID] = expr
		task.NextRunTime = expr.NextTime(time.Now())
	} else if task.Interval > 0 {
		task.NextRunTime = time.Now().Add(task.Interval)
	} else {
		return errors.New("task must have either cron expression or interval")
	}

	s.tasks[task.ID] = task
	s.emitEvent(EventTaskAdded, task, nil)
	s.logger.Printf("Task added: %s (%s)", task.Name, task.ID)

	return nil
}

// RemoveTask 移除任务
func (s *Scheduler) RemoveTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	// 如果任务正在运行，先取消
	if task.GetStatus() == TaskStatusRunning {
		task.Cancel()
	}

	delete(s.tasks, taskID)
	delete(s.cronExpr, taskID)
	s.emitEvent(EventTaskRemoved, task, nil)
	s.logger.Printf("Task removed: %s (%s)", task.Name, task.ID)

	return nil
}

// PauseTask 暂停任务
func (s *Scheduler) PauseTask(taskID string) error {
	s.mu.RLock()
	task, exists := s.tasks[taskID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	task.Pause()
	s.emitEvent(EventTaskPaused, task, nil)
	return nil
}

// ResumeTask 恢复任务
func (s *Scheduler) ResumeTask(taskID string) error {
	s.mu.RLock()
	task, exists := s.tasks[taskID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	task.Resume()
	s.emitEvent(EventTaskResumed, task, nil)
	return nil
}

// GetTask 获取任务
func (s *Scheduler) GetTask(taskID string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task %s not found", taskID)
	}
	return task, nil
}

// ListTasks 列出所有任务
func (s *Scheduler) ListTasks() []TaskInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]TaskInfo, 0, len(s.tasks))
	for _, task := range s.tasks {
		result = append(result, task.Info())
	}
	return result
}

// Start 启动调度器
func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	s.logger.Println("Scheduler started")

	go s.run()
}

// run 调度主循环
func (s *Scheduler) run() {
	ticker := time.NewTicker(s.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkAndRunTasks()
		}
	}
}

// checkAndRunTasks 检查并执行到期的任务
func (s *Scheduler) checkAndRunTasks() {
	now := time.Now()

	s.mu.RLock()
	tasks := make([]*Task, 0)
	for _, task := range s.tasks {
		if s.shouldRunTask(task, now) {
			tasks = append(tasks, task)
		}
	}
	s.mu.RUnlock()

	for _, task := range tasks {
		s.runTask(task)
	}
}

// shouldRunTask 判断任务是否应该执行
func (s *Scheduler) shouldRunTask(task *Task, now time.Time) bool {
	status := task.GetStatus()

	// 只运行等待中的任务
	if status != TaskStatusPending && status != TaskStatusCompleted && status != TaskStatusFailed {
		return false
	}

	// 检查是否到达执行时间
	task.mu.RLock()
	nextRun := task.NextRunTime
	task.mu.RUnlock()

	return !nextRun.IsZero() && !nextRun.After(now)
}

// runTask 执行任务
func (s *Scheduler) runTask(task *Task) {
	// 获取信号量（限制并发）
	select {
	case s.semaphore <- struct{}{}:
	default:
		// 并发数已满，下次再执行
		return
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer func() { <-s.semaphore }()

		// 恢复panic
		defer func() {
			if r := recover(); r != nil {
				s.logger.Printf("Task panic recovered: %s - %v", task.ID, r)
				task.mu.Lock()
				task.Status = TaskStatusFailed
				task.LastError = fmt.Sprintf("panic: %v", r)
				task.mu.Unlock()
			}
		}()

		s.emitEvent(EventTaskStarted, task, nil)
		s.logger.Printf("Task started: %s (%s)", task.Name, task.ID)

		// 执行任务
		err := task.Execute(s.ctx)

		// 更新下次执行时间
		s.updateNextRunTime(task)

		if err != nil {
			s.emitEvent(EventTaskFailed, task, err)
			s.logger.Printf("Task failed: %s (%s) - %v", task.Name, task.ID, err)
		} else {
			s.emitEvent(EventTaskCompleted, task, nil)
			s.logger.Printf("Task completed: %s (%s)", task.Name, task.ID)
		}
	}()
}

// updateNextRunTime 更新下次执行时间
func (s *Scheduler) updateNextRunTime(task *Task) {
	s.mu.RLock()
	cronExpr := s.cronExpr[task.ID]
	s.mu.RUnlock()

	task.mu.Lock()
	defer task.mu.Unlock()

	if cronExpr != nil {
		task.NextRunTime = cronExpr.NextTime(time.Now())
	} else if task.Interval > 0 {
		task.NextRunTime = time.Now().Add(task.Interval)
	}

	// 重置状态为等待
	if task.Status == TaskStatusCompleted || task.Status == TaskStatusFailed {
		task.Status = TaskStatusPending
	}
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	s.logger.Println("Scheduler stopping...")

	// 取消上下文
	s.cancel()

	// 等待所有任务完成
	s.wg.Wait()

	// 关闭事件通道
	close(s.eventCh)

	s.logger.Println("Scheduler stopped")
}

// RunNow 立即执行指定任务
func (s *Scheduler) RunNow(taskID string) error {
	s.mu.RLock()
	task, exists := s.tasks[taskID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	s.runTask(task)
	return nil
}

// Status 获取调度器状态
type SchedulerStatus struct {
	Running       bool      `json:"running"`
	TotalTasks    int       `json:"total_tasks"`
	RunningTasks  int       `json:"running_tasks"`
	PausedTasks   int       `json:"paused_tasks"`
	MaxConcurrent int       `json:"max_concurrent"`
	CurrentTime   time.Time `json:"current_time"`
}

// GetStatus 获取调度器状态
func (s *Scheduler) GetStatus() SchedulerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := SchedulerStatus{
		Running:       s.running,
		TotalTasks:    len(s.tasks),
		MaxConcurrent: s.config.MaxConcurrent,
		CurrentTime:   time.Now(),
	}

	for _, task := range s.tasks {
		switch task.GetStatus() {
		case TaskStatusRunning:
			status.RunningTasks++
		case TaskStatusPaused:
			status.PausedTasks++
		}
	}

	return status
}
