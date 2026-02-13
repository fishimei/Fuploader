package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"Fuploader/internal/database"
	"Fuploader/internal/types"
	"Fuploader/internal/utils"

	"gorm.io/gorm"
)

type EnhancedScheduler struct {
	db         *gorm.DB
	workers    int
	uploaders  map[string]types.Uploader
	taskQueue  chan *database.ScheduledTask
	stopChan   chan struct{}
	wg         sync.WaitGroup
	mu         sync.RWMutex
	running    bool
}

func NewEnhancedScheduler(db *gorm.DB, workers int) *EnhancedScheduler {
	return &EnhancedScheduler{
		db:        db,
		workers:   workers,
		uploaders: make(map[string]types.Uploader),
		taskQueue: make(chan *database.ScheduledTask, 100),
		stopChan:  make(chan struct{}),
	}
}

func (s *EnhancedScheduler) RegisterUploader(platform string, uploader types.Uploader) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.uploaders[platform] = uploader
}

func (s *EnhancedScheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}

	s.wg.Add(1)
	go s.scheduler()

	utils.Info(fmt.Sprintf("[+] 调度器已启动，工作线程数: %d", s.workers))
}

func (s *EnhancedScheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	close(s.stopChan)
	s.wg.Wait()
	close(s.taskQueue)

	utils.Info("[+] 调度器已停止")
}

func (s *EnhancedScheduler) AddTask(task *database.ScheduledTask) error {
	task.Status = database.TaskStatusPending
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()

	if err := s.db.Create(task).Error; err != nil {
		return fmt.Errorf("保存任务失败: %w", err)
	}

	select {
	case s.taskQueue <- task:
		utils.Info(fmt.Sprintf("[+] 任务已添加到队列: %s", task.ID))
		return nil
	default:
		return fmt.Errorf("任务队列已满")
	}
}

func (s *EnhancedScheduler) worker(id int) {
	defer s.wg.Done()

	for {
		select {
		case <-s.stopChan:
			return
		case task, ok := <-s.taskQueue:
			if !ok {
				return
			}
			s.executeTask(task)
		}
	}
}

func (s *EnhancedScheduler) scheduler() {
	defer s.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.checkPendingTasks()
		}
	}
}

func (s *EnhancedScheduler) checkPendingTasks() {
	var tasks []database.ScheduledTask
	now := time.Now()

	if err := s.db.Where("status = ? AND schedule_time <= ?", database.TaskStatusPending, now).
		Order("priority DESC, schedule_time ASC").
		Find(&tasks).Error; err != nil {
		return
	}

	for _, task := range tasks {
		select {
		case s.taskQueue <- &task:
		default:
		}
	}
}

func (s *EnhancedScheduler) executeTask(task *database.ScheduledTask) {
	s.mu.RLock()
	uploader, ok := s.uploaders[task.Platform]
	s.mu.RUnlock()

	if !ok {
		s.updateTaskStatus(task, database.TaskStatusFailed, "未找到平台上传器")
		return
	}

	s.updateTaskStatus(task, database.TaskStatusRunning, "")

	videoTask := &types.VideoTask{
		VideoPath:   task.VideoPath,
		Title:       task.Title,
		Description: task.Description,
	}

	ctx := context.Background()
	if err := uploader.Upload(ctx, videoTask); err != nil {
		s.updateTaskStatus(task, database.TaskStatusFailed, err.Error())
		return
	}

	s.updateTaskStatus(task, database.TaskStatusCompleted, "")
}

func (s *EnhancedScheduler) updateTaskStatus(task *database.ScheduledTask, status database.TaskStatus, errMsg string) {
	task.Status = status
	task.Error = errMsg
	task.UpdatedAt = time.Now()

	if status == database.TaskStatusCompleted {
		now := time.Now()
		task.CompletedAt = &now
	}

	s.db.Save(task)
}
