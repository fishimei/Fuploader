package scheduler

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"Fuploader/internal/database"
	"Fuploader/internal/types"
	"Fuploader/internal/utils"

	"gorm.io/gorm"
)

// EnhancedScheduler 增强调度器
type EnhancedScheduler struct {
	db           *gorm.DB
	taskQueue    chan *database.ScheduledTask
	workers      int
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	uploaderMap  map[string]types.Uploader
	mutex        sync.RWMutex
	runningTasks map[string]context.CancelFunc
}

// NewEnhancedScheduler 创建增强调度器
func NewEnhancedScheduler(db *gorm.DB, workers int) *EnhancedScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &EnhancedScheduler{
		db:           db,
		taskQueue:    make(chan *database.ScheduledTask, 100),
		workers:      workers,
		ctx:          ctx,
		cancel:       cancel,
		uploaderMap:  make(map[string]types.Uploader),
		runningTasks: make(map[string]context.CancelFunc),
	}
}

// RegisterUploader 注册上传器
func (s *EnhancedScheduler) RegisterUploader(platform string, uploader types.Uploader) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.uploaderMap[platform] = uploader
}

// Start 启动调度器
func (s *EnhancedScheduler) Start() {
	// 启动工作线程
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}

	// 启动任务扫描线程
	go s.scanLoop()

	utils.Info("[+] 增强调度器已启动")
}

// Stop 停止调度器
func (s *EnhancedScheduler) Stop() {
	s.cancel()

	// 取消所有运行中的任务
	s.mutex.Lock()
	for _, cancel := range s.runningTasks {
		cancel()
	}
	s.mutex.Unlock()

	close(s.taskQueue)
	s.wg.Wait()

	utils.Info("[-] 增强调度器已停止")
}

// AddTask 添加任务
func (s *EnhancedScheduler) AddTask(task *database.ScheduledTask) error {
	task.Status = database.TaskStatusPending
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()

	if task.MaxRetries == 0 {
		task.MaxRetries = 3
	}

	// 保存到数据库
	if err := s.db.Create(task).Error; err != nil {
		return fmt.Errorf("save task to db failed: %w", err)
	}

	// 如果任务时间已到，立即放入队列
	if time.Now().After(task.ScheduleTime) {
		s.taskQueue <- task
	}

	utils.Info(fmt.Sprintf("[+] 任务已添加 - ID: %s, 平台: %s", task.ID, task.Platform))
	return nil
}

// CancelTask 取消任务
func (s *EnhancedScheduler) CancelTask(taskID string) error {
	s.mutex.Lock()
	if cancel, ok := s.runningTasks[taskID]; ok {
		cancel()
		delete(s.runningTasks, taskID)
	}
	s.mutex.Unlock()

	return s.db.Model(&database.ScheduledTask{}).
		Where("id = ?", taskID).
		Update("status", database.TaskStatusCancelled).Error
}

// GetTaskStatus 获取任务状态
func (s *EnhancedScheduler) GetTaskStatus(taskID string) (*database.ScheduledTask, error) {
	var task database.ScheduledTask
	if err := s.db.Where("id = ?", taskID).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// ListTasks 列出任务
func (s *EnhancedScheduler) ListTasks(status database.TaskStatus, limit int) ([]*database.ScheduledTask, error) {
	var tasks []*database.ScheduledTask
	query := s.db
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Order("created_at DESC").Limit(limit).Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// worker 工作线程
func (s *EnhancedScheduler) worker(id int) {
	defer s.wg.Done()

	utils.Info(fmt.Sprintf("[-] 工作线程 %d 已启动", id))

	for task := range s.taskQueue {
		if task.Status == database.TaskStatusCancelled {
			continue
		}

		s.executeTask(task)
	}

	utils.Info(fmt.Sprintf("[-] 工作线程 %d 已停止", id))
}

// executeTask 执行任务
func (s *EnhancedScheduler) executeTask(task *database.ScheduledTask) {
	// 创建任务上下文
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Minute)
	defer cancel()

	// 记录运行中的任务
	s.mutex.Lock()
	s.runningTasks[task.ID] = cancel
	s.mutex.Unlock()

	defer func() {
		s.mutex.Lock()
		delete(s.runningTasks, task.ID)
		s.mutex.Unlock()
	}()

	// 更新任务状态
	task.Status = database.TaskStatusRunning
	now := time.Now()
	task.ExecutedAt = &now
	s.updateTask(task)

	utils.Info(fmt.Sprintf("[+] 开始执行任务 - ID: %s, 平台: %s", task.ID, task.Platform))

	// 获取上传器
	s.mutex.RLock()
	uploader, ok := s.uploaderMap[task.Platform]
	s.mutex.RUnlock()

	if !ok {
		s.failTask(task, fmt.Sprintf("uploader not found for platform: %s", task.Platform))
		return
	}

	// 构建 VideoTask
	// 使用任务标题（如果有），否则使用视频文件名（去掉扩展名）
	title := task.Title
	if title == "" {
		title = filepath.Base(task.VideoPath)
		// 去掉扩展名
		if ext := filepath.Ext(title); ext != "" {
			title = title[:len(title)-len(ext)]
		}
	}

	// 将 ScheduleTime 转换为字符串格式
	scheduleTimeStr := task.ScheduleTime.Format("2006-01-02 15:04")

	videoTask := &types.VideoTask{
		VideoPath:    task.VideoPath,
		Title:        title,
		Description:  task.Description,
		ScheduleTime: &scheduleTimeStr,
	}
	// 解析 Tags
	if task.Tags != "" {
		videoTask.Tags = strings.Split(task.Tags, ",")
	}

	// 执行上传
	err := uploader.Upload(ctx, videoTask)

	if err != nil {
		// 检查是否需要重试
		if task.RetryCount < task.MaxRetries {
			task.RetryCount++
			task.Status = database.TaskStatusPending
			task.Error = err.Error()
			s.updateTask(task)

			utils.Warn(fmt.Sprintf("[-] 任务失败，准备重试 - ID: %s, 重试次数: %d", task.ID, task.RetryCount))

			// 延迟后重新加入队列
			go func() {
				time.Sleep(time.Duration(task.RetryCount) * 5 * time.Minute)
				s.taskQueue <- task
			}()
			return
		}

		s.failTask(task, err.Error())
		return
	}

	// 任务成功
	completedAt := time.Now()
	task.Status = database.TaskStatusCompleted
	task.CompletedAt = &completedAt
	task.Result = "upload completed"
	s.updateTask(task)

	utils.Info(fmt.Sprintf("[+] 任务完成 - ID: %s, 平台: %s", task.ID, task.Platform))
}

// failTask 标记任务失败
func (s *EnhancedScheduler) failTask(task *database.ScheduledTask, errorMsg string) {
	task.Status = database.TaskStatusFailed
	task.Error = errorMsg
	s.updateTask(task)

	utils.Error(fmt.Sprintf("[-] 任务失败 - ID: %s, 错误: %s", task.ID, errorMsg))
}

// updateTask 更新任务
func (s *EnhancedScheduler) updateTask(task *database.ScheduledTask) {
	task.UpdatedAt = time.Now()
	s.db.Save(task)
}

// scanLoop 扫描循环
func (s *EnhancedScheduler) scanLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.scanPendingTasks()
		}
	}
}

// scanPendingTasks 扫描待执行任务
func (s *EnhancedScheduler) scanPendingTasks() {
	var tasks []database.ScheduledTask

	// 查询到期的任务
	if err := s.db.Where("status = ? AND schedule_time <= ?",
		database.TaskStatusPending, time.Now()).
		Order("priority DESC, schedule_time ASC").
		Find(&tasks).Error; err != nil {
		utils.Warn(fmt.Sprintf("[-] 扫描待执行任务失败: %v", err))
		return
	}

	for i := range tasks {
		task := &tasks[i]
		task.Status = database.TaskStatusScheduled
		s.updateTask(task)
		s.taskQueue <- task

		utils.Info(fmt.Sprintf("[+] 任务加入队列 - ID: %s, 平台: %s", task.ID, task.Platform))
	}
}
