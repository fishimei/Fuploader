package service

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"Fuploader/internal/config"
	"Fuploader/internal/database"
	"Fuploader/internal/platform/baijiahao"
	"Fuploader/internal/platform/bilibili"
	"Fuploader/internal/platform/douyin"
	"Fuploader/internal/platform/kuaishou"
	"Fuploader/internal/platform/ratelimit"
	"Fuploader/internal/platform/tencent"
	"Fuploader/internal/platform/tiktok"
	"Fuploader/internal/platform/xiaohongshu"
	"Fuploader/internal/types"
	"Fuploader/internal/utils"

	"gorm.io/gorm"
)

type UploadService struct {
	db          *gorm.DB
	eventBus    *EventBus
	rateLimiter *ratelimit.LimiterWithStats
}

// EventHandler 事件处理器函数类型
type EventHandler func(event types.Event)

type EventBus struct {
	handlers map[string][]EventHandler
}

// UploadTaskMetadata 上传任务元数据（使用types包中的定义）
type UploadTaskMetadata = types.UploadTaskMetadata

func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[string][]EventHandler),
	}
}

func (eb *EventBus) Subscribe(event string, handler EventHandler) {
	eb.handlers[event] = append(eb.handlers[event], handler)
}

func (eb *EventBus) Publish(event string, data types.Event) {
	for _, handler := range eb.handlers[event] {
		go handler(data)
	}
}

func NewUploadService(db *gorm.DB) *UploadService {
	return &UploadService{
		db:          db,
		eventBus:    NewEventBus(),
		rateLimiter: ratelimit.NewLimiterWithStats(),
	}
}

func (s *UploadService) GetEventBus() *EventBus {
	return s.eventBus
}

// checkRateLimit 检查平台限流
func (s *UploadService) checkRateLimit(platform string) error {
	// 检查令牌桶限流
	if !s.rateLimiter.Allow(platform) {
		return fmt.Errorf("platform %s rate limit exceeded, please try again later", platform)
	}

	// 检查每日/每小时上传限制
	var dailyCount, hourlyCount int64
	today := time.Now().Format("2006-01-02")
	hour := time.Now().Format("2006-01-02 15")

	s.db.Model(&database.UploadTask{}).
		Where("platform = ? AND DATE(created_at) = ? AND status = ?", platform, today, config.TaskStatusSuccess).
		Count(&dailyCount)

	s.db.Model(&database.UploadTask{}).
		Where("platform = ? AND DATE_FORMAT(created_at, '%Y-%m-%d %H') = ? AND status = ?", platform, hour, config.TaskStatusSuccess).
		Count(&hourlyCount)

	if err := s.rateLimiter.CheckUploadLimit(platform, int(dailyCount), int(hourlyCount)); err != nil {
		return err
	}

	return nil
}

func (s *UploadService) CreateUploadTask(ctx context.Context, videoID int, accountIDs []int, scheduleTime *string, metadata *UploadTaskMetadata) ([]database.UploadTask, error) {
	var video database.Video
	if result := s.db.First(&video, videoID); result.Error != nil {
		return nil, fmt.Errorf("video not found")
	}

	var tasks []database.UploadTask
	for _, accountID := range accountIDs {
		var account database.Account
		if result := s.db.First(&account, accountID); result.Error != nil {
			continue
		}

		// 检查限流
		if err := s.checkRateLimit(account.Platform); err != nil {
			utils.Warn(fmt.Sprintf("[-] 平台 %s 限流检查失败: %v", account.Platform, err))
			continue
		}

		// 无论是否有定时时间，都立即执行任务（由平台处理定时发布）
		task := database.UploadTask{
			VideoID:      videoID,
			AccountID:    accountID,
			Platform:     account.Platform,
			Status:       config.TaskStatusUploading,
			Progress:     0,
			ScheduleTime: scheduleTime,
		}

		// 应用通用标题（如果用户填写了）
		if metadata != nil && metadata.Common.Title != "" {
			task.Title = metadata.Common.Title
		}

		// 应用平台特定字段
		if metadata != nil && metadata.Platforms != nil {
			if platformFields, ok := metadata.Platforms[account.Platform]; ok {
				// 应用平台特定字段到任务
				s.applyPlatformFields(&task, platformFields)
			}
		}

		result := s.db.Create(&task)
		if result.Error != nil {
			utils.Error(fmt.Sprintf("Create task failed: %v", result.Error))
			continue
		}

		tasks = append(tasks, task)

		// 立即执行上传任务（平台会处理定时发布逻辑）
		go s.executeTask(context.Background(), task.ID)
	}

	return tasks, nil
}

// applyPlatformFields 应用平台特定字段到任务
func (s *UploadService) applyPlatformFields(task *database.UploadTask, fields types.PlatformFields) {
	if fields.Title != "" {
		task.Title = fields.Title
	}
	if fields.Collection != "" {
		task.Collection = fields.Collection
	}
	if fields.ShortTitle != "" {
		task.ShortTitle = fields.ShortTitle
	}
	task.IsOriginal = fields.IsOriginal
	task.OriginalType = fields.OriginalType
	if fields.Location != "" {
		task.Location = fields.Location
	}
	if fields.Thumbnail != "" {
		task.Thumbnail = fields.Thumbnail
	}
	task.SyncToutiao = fields.SyncToutiao
	task.SyncXigua = fields.SyncXigua
	task.IsDraft = fields.IsDraft
	if fields.Copyright != "" {
		task.Copyright = fields.Copyright
	}
	task.AllowDownload = fields.AllowDownload
	task.AllowComment = fields.AllowComment
	task.AllowDuet = fields.AllowDuet
	task.AIDeclaration = fields.AIDeclaration
	task.AutoGenerateAudio = fields.AutoGenerateAudio
	task.CoverType = fields.CoverType
	task.Category = fields.Category
	task.UseIframe = fields.UseIframe
	task.UseFileChooser = fields.UseFileChooser
	task.SkipNewFeatureGuide = fields.SkipNewFeatureGuide
}

func (s *UploadService) GetUploadTasks(ctx context.Context, status string) ([]database.UploadTask, error) {
	var tasks []database.UploadTask
	query := s.db.Preload("Video").Preload("Account")

	if status != "" {
		query = query.Where("status = ?", status)
	}

	result := query.Find(&tasks)
	if result.Error != nil {
		return nil, fmt.Errorf("query tasks failed: %w", result.Error)
	}
	return tasks, nil
}

func (s *UploadService) CancelUploadTask(ctx context.Context, id int) error {
	var task database.UploadTask
	result := s.db.First(&task, id)
	if result.Error != nil {
		return fmt.Errorf("task not found")
	}

	oldStatus := task.Status

	if task.Status != config.TaskStatusPending && task.Status != config.TaskStatusUploading {
		return fmt.Errorf("task cannot be cancelled")
	}

	task.Status = config.TaskStatusCancelled
	result = s.db.Save(&task)
	if result.Error != nil {
		return fmt.Errorf("cancel task failed: %w", result.Error)
	}

	s.eventBus.Publish(config.EventTaskStatusChanged, types.TaskStatusChangedEvent{
		TaskID:    id,
		OldStatus: oldStatus,
		NewStatus: config.TaskStatusCancelled,
	})

	return nil
}

func (s *UploadService) RetryUploadTask(ctx context.Context, id int) error {
	var task database.UploadTask
	result := s.db.First(&task, id)
	if result.Error != nil {
		return fmt.Errorf("task not found")
	}

	if task.Status != config.TaskStatusFailed {
		return fmt.Errorf("only failed tasks can be retried")
	}

	// 检查限流
	if err := s.checkRateLimit(task.Platform); err != nil {
		return fmt.Errorf("rate limit check failed: %w", err)
	}

	task.Status = config.TaskStatusUploading
	task.RetryCount++
	task.ErrorMsg = ""
	result = s.db.Save(&task)
	if result.Error != nil {
		return fmt.Errorf("retry task failed: %w", result.Error)
	}

	go s.executeTask(context.Background(), id)

	return nil
}

func (s *UploadService) DeleteUploadTask(ctx context.Context, id int) error {
	result := s.db.Delete(&database.UploadTask{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete task failed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("task not found")
	}
	return nil
}

// GetRateLimitStats 获取限流统计
func (s *UploadService) GetRateLimitStats(platform string) (*ratelimit.Stats, bool) {
	return s.rateLimiter.GetStats(platform)
}

// GetAllRateLimitStats 获取所有限流统计
func (s *UploadService) GetAllRateLimitStats() map[string]ratelimit.Stats {
	return s.rateLimiter.GetAllStats()
}

func (s *UploadService) executeTask(ctx context.Context, taskID int) {
	var task database.UploadTask
	if result := s.db.Preload("Video").Preload("Account").First(&task, taskID); result.Error != nil {
		utils.Error(fmt.Sprintf("Task %d not found", taskID))
		return
	}

	// 记录开始上传日志
	s.createUploadLog(taskID, "upload_start", "开始上传")

	s.eventBus.Publish(config.EventUploadProgress, types.UploadProgressEvent{
		TaskID:   task.ID,
		Platform: task.Platform,
		Progress: 10,
		Message:  "Starting upload...",
	})

	// 使用用户自定义标题（如果有），否则使用视频标题，最后使用文件名作为默认
	title := task.Video.Title
	if task.Title != "" {
		title = task.Title
	}
	// 如果标题仍为空，使用文件名（去掉扩展名）
	if title == "" {
		title = filepath.Base(task.Video.FilePath)
		// 去掉扩展名
		if ext := filepath.Ext(title); ext != "" {
			title = title[:len(title)-len(ext)]
		}
	}

	// 封面优先级：发布页面设置的封面 > 视频默认封面
	thumbnail := task.Thumbnail
	if thumbnail == "" {
		thumbnail = task.Video.Thumbnail
	}

	// 将 URL 路径转换为本地文件系统路径
	thumbnail = convertThumbnailURLToPath(thumbnail)

	videoTask := &types.VideoTask{
		Platform:            task.Platform,
		VideoPath:           task.Video.FilePath,
		Title:               title,
		Description:         task.Video.Description,
		Tags:                task.Video.Tags,
		Thumbnail:           thumbnail,
		ScheduleTime:        task.ScheduleTime,
		IsDraft:             task.IsDraft,
		Location:            task.Location,
		SyncToutiao:         task.SyncToutiao,
		SyncXigua:           task.SyncXigua,
		ShortTitle:          task.ShortTitle,
		IsOriginal:          task.IsOriginal,
		OriginalType:        task.OriginalType,
		Collection:          task.Collection,
		Copyright:           task.Copyright,
		AllowDownload:       task.AllowDownload,
		AllowComment:        task.AllowComment,
		AllowDuet:           task.AllowDuet,
		AIDeclaration:       task.AIDeclaration,
		AutoGenerateAudio:   task.AutoGenerateAudio,
		CoverType:           task.CoverType,
		Category:            task.Category,
		UseIframe:           task.UseIframe,
		UseFileChooser:      task.UseFileChooser,
		SkipNewFeatureGuide: task.SkipNewFeatureGuide,
	}

	var uploader types.Uploader
	switch task.Platform {
	case config.PlatformDouyin:
		uploader = douyin.NewUploader(task.Account.CookiePath)
	case config.PlatformTencent:
		uploader = tencent.NewUploaderWithAccount(uint(task.AccountID))
	case config.PlatformKuaishou:
		uploader = kuaishou.NewUploader(task.Account.CookiePath)
	case config.PlatformTiktok:
		uploader = tiktok.NewUploader(task.Account.CookiePath)
	case config.PlatformXiaohongshu:
		uploader = xiaohongshu.NewUploader(task.Account.CookiePath)
	case config.PlatformBaijiahao:
		uploader = baijiahao.NewUploader(task.Account.CookiePath)
	case config.PlatformBilibili:
		uploader = bilibili.NewUploader(task.Account.CookiePath)
	default:
		s.updateTaskFailed(taskID, "unsupported platform")
		s.createUploadLog(taskID, "upload_error", "不支持的平台")
		s.eventBus.Publish(config.EventUploadError, types.UploadErrorEvent{
			TaskID:   task.ID,
			Platform: task.Platform,
			Error:    "unsupported platform",
			CanRetry: false,
		})
		return
	}

	s.eventBus.Publish(config.EventUploadProgress, types.UploadProgressEvent{
		TaskID:   task.ID,
		Platform: task.Platform,
		Progress: 30,
		Message:  "Uploading video...",
	})

	err := uploader.Upload(ctx, videoTask)
	if err != nil {
		s.updateTaskFailed(taskID, err.Error())
		s.createUploadLog(taskID, "upload_error", "上传失败: "+err.Error())
		s.eventBus.Publish(config.EventUploadError, types.UploadErrorEvent{
			TaskID:   task.ID,
			Platform: task.Platform,
			Error:    err.Error(),
			CanRetry: true,
		})
		return
	}

	// 验证发布结果
	s.eventBus.Publish(config.EventUploadProgress, types.UploadProgressEvent{
		TaskID:   task.ID,
		Platform: task.Platform,
		Progress: 95,
		Message:  "验证发布结果...",
	})

	// 检查任务是否仍然有效（没有被取消）
	var currentTask database.UploadTask
	if result := s.db.First(&currentTask, taskID); result.Error != nil {
		s.updateTaskFailed(taskID, "任务不存在")
		return
	}

	if currentTask.Status == config.TaskStatusCancelled {
		utils.Warn(fmt.Sprintf("[-] 任务 %d 已被取消，跳过成功处理", taskID))
		return
	}

	task.Status = config.TaskStatusSuccess
	task.Progress = 100
	if err := s.db.Save(&task).Error; err != nil {
		utils.Error(fmt.Sprintf("[-] 保存任务状态失败: %v", err))
		s.updateTaskFailed(taskID, "保存任务状态失败: "+err.Error())
		return
	}

	s.createUploadLog(taskID, "upload_success", "上传成功")

	s.eventBus.Publish(config.EventUploadComplete, types.UploadCompleteEvent{
		TaskID:      task.ID,
		Platform:    task.Platform,
		PublishURL:  task.PublishURL,
		CompletedAt: time.Now().Format(time.RFC3339),
	})

	s.eventBus.Publish(config.EventTaskStatusChanged, types.TaskStatusChangedEvent{
		TaskID:    taskID,
		OldStatus: config.TaskStatusUploading,
		NewStatus: config.TaskStatusSuccess,
	})
}

// createUploadLog 创建上传日志
func (s *UploadService) createUploadLog(taskID int, step, message string) {
	log := database.UploadLog{
		TaskID:  uint(taskID),
		Step:    step,
		Message: message,
		Status:  "processing",
	}
	if err := s.db.Create(&log).Error; err != nil {
		utils.Warn(fmt.Sprintf("[-] 创建上传日志失败: %v", err))
	}
}

func (s *UploadService) updateTaskFailed(taskID int, errorMsg string) {
	var task database.UploadTask
	if result := s.db.First(&task, taskID); result.Error != nil {
		return
	}
	task.Status = config.TaskStatusFailed
	task.ErrorMsg = errorMsg
	s.db.Save(&task)
}

// convertThumbnailURLToPath 将封面 URL 路径转换为本地文件系统路径
// 例如: /thumbnails/thumb_1_123.jpg -> D:\storage\thumbnails\thumb_1_123.jpg
func convertThumbnailURLToPath(thumbnail string) string {
	if thumbnail == "" {
		return ""
	}

	// 如果已经是本地路径（包含盘符或绝对路径），直接返回
	if filepath.IsAbs(thumbnail) {
		return thumbnail
	}

	// 如果是 URL 路径（以 /thumbnails/ 开头），转换为本地路径
	if len(thumbnail) > 12 && thumbnail[:12] == "/thumbnails/" {
		// 提取文件名
		filename := filepath.Base(thumbnail)
		return filepath.Join(config.Config.ThumbnailPath, filename)
	}

	// 如果是 URL 路径（以 /videos/ 开头），转换为本地路径
	if len(thumbnail) > 8 && thumbnail[:8] == "/videos/" {
		filename := filepath.Base(thumbnail)
		return filepath.Join(config.Config.VideoPath, filename)
	}

	// 其他情况，假设是相对路径，拼接到缩略图目录
	return filepath.Join(config.Config.ThumbnailPath, filepath.Base(thumbnail))
}
