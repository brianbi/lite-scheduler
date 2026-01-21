package lite_scheduler

import (
	"encoding/json"
	"os"
	"sync"
)

// TaskStore 任务存储接口
type TaskStore interface {
	Save(tasks []*Task) error
	Load() ([]*Task, error)
}

// FileStore 文件存储
type FileStore struct {
	filepath string
	mu       sync.Mutex
}

// TaskData 任务持久化数据
type TaskData struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CronExpr    string `json:"cron_expr"`
	Interval    int64  `json:"interval_ns"`
	Timeout     int64  `json:"timeout_ns"`
	MaxRetries  int    `json:"max_retries"`
	RetryDelay  int64  `json:"retry_delay_ns"`
}

// NewFileStore 创建文件存储
func NewFileStore(filepath string) *FileStore {
	return &FileStore{filepath: filepath}
}

// Save 保存任务
func (fs *FileStore) Save(tasks []*Task) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data := make([]TaskData, 0, len(tasks))
	for _, task := range tasks {
		task.mu.RLock()
		data = append(data, TaskData{
			ID:          task.ID,
			Name:        task.Name,
			Description: task.Description,
			CronExpr:    task.CronExpr,
			Interval:    int64(task.Interval),
			Timeout:     int64(task.Timeout),
			MaxRetries:  task.MaxRetries,
			RetryDelay:  int64(task.RetryDelay),
		})
		task.mu.RUnlock()
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(fs.filepath, jsonData, 0644)
}

// Load 加载任务（注意：需要重新绑定任务函数）
func (fs *FileStore) Load() ([]TaskData, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data, err := os.ReadFile(fs.filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var tasks []TaskData
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}
