package lite_scheduler

import (
	"context"
	"sync"
	"time"
)

// TaskStatus 任务状态
type TaskStatus int

const (
	TaskStatusPending   TaskStatus = iota // 等待中
	TaskStatusRunning                     // 运行中
	TaskStatusCompleted                   // 已完成
	TaskStatusFailed                      // 失败
	TaskStatusCancelled                   // 已取消
	TaskStatusPaused                      // 已暂停
)

func (s TaskStatus) String() string {
	names := []string{"Pending", "Running", "Completed", "Failed", "Cancelled", "Paused"}
	if int(s) < len(names) {
		return names[s]
	}
	return "Unknown"
}

// TaskFunc 任务执行函数类型
type TaskFunc func(ctx context.Context) error

// Task 定时任务
type Task struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	CronExpr    string        `json:"cron_expr"`   // Cron表达式
	Interval    time.Duration `json:"interval"`    // 固定间隔（与CronExpr二选一）
	Timeout     time.Duration `json:"timeout"`     // 执行超时
	RetryCount  int           `json:"retry_count"` // 重试次数
	RetryDelay  time.Duration `json:"retry_delay"` // 重试延迟
	MaxRetries  int           `json:"max_retries"` // 最大重试次数

	// 运行时状态
	Status      TaskStatus `json:"status"`
	LastRunTime time.Time  `json:"last_run_time"`
	NextRunTime time.Time  `json:"next_run_time"`
	RunCount    int64      `json:"run_count"`
	FailCount   int64      `json:"fail_count"`
	LastError   string     `json:"last_error"`
	CreatedAt   time.Time  `json:"created_at"`

	// 内部字段
	fn         TaskFunc
	cancelFunc context.CancelFunc
	mu         sync.RWMutex
}

// NewTask 创建新任务
func NewTask(id, name string, fn TaskFunc) *Task {
	return &Task{
		ID:         id,
		Name:       name,
		fn:         fn,
		Status:     TaskStatusPending,
		MaxRetries: 3,
		RetryDelay: time.Second * 5,
		Timeout:    time.Minute * 5,
		CreatedAt:  time.Now(),
	}
}

// WithCron 设置Cron表达式
func (t *Task) WithCron(expr string) *Task {
	t.CronExpr = expr
	return t
}

// WithInterval 设置固定间隔
func (t *Task) WithInterval(d time.Duration) *Task {
	t.Interval = d
	return t
}

// WithTimeout 设置超时时间
func (t *Task) WithTimeout(d time.Duration) *Task {
	t.Timeout = d
	return t
}

// WithRetry 设置重试策略
func (t *Task) WithRetry(maxRetries int, delay time.Duration) *Task {
	t.MaxRetries = maxRetries
	t.RetryDelay = delay
	return t
}

// Execute 执行任务
func (t *Task) Execute(ctx context.Context) error {
	t.mu.Lock()
	t.Status = TaskStatusRunning
	t.LastRunTime = time.Now()
	t.mu.Unlock()

	// 创建带超时的上下文
	execCtx, cancel := context.WithTimeout(ctx, t.Timeout)
	defer cancel()

	t.mu.Lock()
	t.cancelFunc = cancel
	t.mu.Unlock()

	var lastErr error
	for attempt := 0; attempt <= t.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-execCtx.Done():
				return execCtx.Err()
			case <-time.After(t.RetryDelay):
			}
		}

		if err := t.fn(execCtx); err != nil {
			lastErr = err
			t.mu.Lock()
			t.RetryCount = attempt + 1
			t.mu.Unlock()
			continue
		}

		// 执行成功
		t.mu.Lock()
		t.Status = TaskStatusCompleted
		t.RunCount++
		t.RetryCount = 0
		t.LastError = ""
		t.mu.Unlock()
		return nil
	}

	// 所有重试都失败
	t.mu.Lock()
	t.Status = TaskStatusFailed
	t.FailCount++
	t.LastError = lastErr.Error()
	t.mu.Unlock()

	return lastErr
}

// Cancel 取消任务
func (t *Task) Cancel() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cancelFunc != nil {
		t.cancelFunc()
	}
	t.Status = TaskStatusCancelled
}

// Pause 暂停任务
func (t *Task) Pause() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Status = TaskStatusPaused
}

// Resume 恢复任务
func (t *Task) Resume() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.Status == TaskStatusPaused {
		t.Status = TaskStatusPending
	}
}

// GetStatus 获取任务状态
func (t *Task) GetStatus() TaskStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Status
}

// TaskInfo 任务信息（用于展示）
type TaskInfo struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Status      string        `json:"status"`
	CronExpr    string        `json:"cron_expr"`
	Interval    time.Duration `json:"interval"`
	LastRunTime time.Time     `json:"last_run_time"`
	NextRunTime time.Time     `json:"next_run_time"`
	RunCount    int64         `json:"run_count"`
	FailCount   int64         `json:"fail_count"`
	LastError   string        `json:"last_error"`
}

// Info 获取任务信息
func (t *Task) Info() TaskInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return TaskInfo{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		Status:      t.Status.String(),
		CronExpr:    t.CronExpr,
		Interval:    t.Interval,
		LastRunTime: t.LastRunTime,
		NextRunTime: t.NextRunTime,
		RunCount:    t.RunCount,
		FailCount:   t.FailCount,
		LastError:   t.LastError,
	}
}
