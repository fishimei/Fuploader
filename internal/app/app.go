package app

import (
	"Fuploader/internal/config"
	"Fuploader/internal/database"
	"Fuploader/internal/platform/baijiahao"
	"Fuploader/internal/platform/bilibili"
	"Fuploader/internal/platform/douyin"
	"Fuploader/internal/platform/kuaishou"
	"Fuploader/internal/platform/tiktok"
	"Fuploader/internal/platform/xiaohongshu"
	"Fuploader/internal/scheduler"
	"Fuploader/internal/service"
	"Fuploader/internal/types"
	"Fuploader/internal/utils"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

type App struct {
	ctx               context.Context
	accountService    *service.AccountService
	fileService       *service.FileService
	uploadService     *service.UploadService
	scheduleService   *service.ScheduleService
	logService        *service.LogService
	screenshotService *service.ScreenshotService
	scheduler         *scheduler.EnhancedScheduler
	initialized       bool
	initError         string
}

func NewApp() *App {
	return &App{}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	if err := config.Init(); err != nil {
		a.initError = fmt.Sprintf("Config init failed: %v", err)
		fmt.Println(a.initError)
		return
	}

	if err := utils.InitLogger(); err != nil {
		a.initError = fmt.Sprintf("Logger init failed: %v", err)
		fmt.Println(a.initError)
		return
	}

	if err := database.Init(); err != nil {
		a.initError = fmt.Sprintf("Database init failed: %v", err)
		fmt.Println(a.initError)
		return
	}

	db := database.GetDB()
	a.accountService = service.NewAccountService(db)
	a.fileService = service.NewFileService(db)
	a.uploadService = service.NewUploadService(db)
	a.scheduleService = service.NewScheduleService(db)
	a.logService = service.NewLogService()
	a.screenshotService = service.NewScreenshotService()

	// 将 LogService 注入到 logger，使日志同时输出到前端
	utils.SetLogService(a.logService)

	a.setupEventListeners()

	// 设置并启动增强调度器
	a.setupScheduler(db)

	a.initialized = true
	utils.Info("Application started successfully")
}

// setupScheduler 设置调度器
func (a *App) setupScheduler(db *gorm.DB) {
	// 创建调度器，使用5个工作线程
	a.scheduler = scheduler.NewEnhancedScheduler(db, 5)

	// 注册各平台上传器
	a.registerUploaders()

	// 启动调度器
	a.scheduler.Start()

	utils.Info("[+] 调度器已启动")
}

// registerUploaders 注册各平台上传器
func (a *App) registerUploaders() {
	// 获取所有账号，为每个账号创建上传器
	accounts, err := a.accountService.GetAccounts(a.ctx)
	if err != nil {
		utils.Error(fmt.Sprintf("[-] 获取账号列表失败: %v", err))
		return
	}

	for _, account := range accounts {
		cookiePath := a.accountService.GetCookiePath(account.Platform, uint(account.ID))
		var uploader types.Uploader

		switch account.Platform {
		case "xiaohongshu":
			uploader = xiaohongshu.NewUploader(cookiePath)
		case "douyin":
			uploader = douyin.NewUploader(cookiePath)
		case "bilibili":
			uploader = bilibili.NewUploader(cookiePath)
		case "kuaishou":
			uploader = kuaishou.NewUploader(cookiePath)
		case "tiktok":
			uploader = tiktok.NewUploader(cookiePath)
		case "baijiahao":
			uploader = baijiahao.NewUploader(cookiePath)
		default:
			utils.Warn(fmt.Sprintf("[-] 未知平台: %s", account.Platform))
			continue
		}

		if uploader != nil {
			a.scheduler.RegisterUploader(account.Platform, uploader)
			utils.Info(fmt.Sprintf("[+] 已注册上传器 - 平台: %s, 账号: %s", account.Platform, account.Name))
		}
	}
}

// GetAppStatus 获取应用初始化状态
func (a *App) GetAppStatus() (*types.AppStatus, error) {
	return &types.AppStatus{
		Initialized: a.initialized,
		Error:       a.initError,
		Version:     config.AppVersion,
	}, nil
}

func (a *App) Shutdown(ctx context.Context) {
	utils.Info("Application is shutting down...")

	// 优雅关闭：等待运行中的任务完成
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// 停止调度器
	if a.scheduler != nil {
		a.scheduler.Stop()
		utils.Info("[-] 调度器已停止")
	}

	// 等待所有任务完成或超时
	select {
	case <-shutdownCtx.Done():
		utils.Warn("[-] 关闭超时，强制退出")
	case <-time.After(1 * time.Second):
		// 给一点时间让日志写入
	}

	// 关闭数据库
	if err := database.Close(); err != nil {
		utils.Error(fmt.Sprintf("[-] 数据库关闭失败: %v", err))
	}

	utils.Info("Application shutdown complete")
}

// ExecuteTask 执行指定任务（由调度器调用）
func (a *App) ExecuteTask(taskID int) error {
	// 调度器会调用此方法执行定时任务
	// 实际执行逻辑在 uploadService.executeTask 中
	// 这里触发事件通知前端
	a.emitEvent(config.EventTaskStatusChanged, types.TaskStatusChangedEvent{
		TaskID:    taskID,
		OldStatus: config.TaskStatusPending,
		NewStatus: config.TaskStatusUploading,
	})
	return nil
}

func (a *App) emitEvent(eventName string, data types.Event) {
	if a.ctx == nil {
		return
	}
	wailsRuntime.EventsEmit(a.ctx, eventName, data)
}

func (a *App) setupEventListeners() {
	eventBus := a.uploadService.GetEventBus()

	eventBus.Subscribe(config.EventUploadProgress, func(data types.Event) {
		if progress, ok := data.(types.UploadProgressEvent); ok {
			a.emitEvent(config.EventUploadProgress, progress)
		}
	})

	eventBus.Subscribe(config.EventUploadComplete, func(data types.Event) {
		if result, ok := data.(types.UploadCompleteEvent); ok {
			a.emitEvent(config.EventUploadComplete, result)
		}
	})

	eventBus.Subscribe(config.EventUploadError, func(data types.Event) {
		if result, ok := data.(types.UploadErrorEvent); ok {
			a.emitEvent(config.EventUploadError, result)
		}
	})

	eventBus.Subscribe(config.EventTaskStatusChanged, func(data types.Event) {
		if eventData, ok := data.(types.TaskStatusChangedEvent); ok {
			a.emitEvent(config.EventTaskStatusChanged, eventData)
		}
	})
}

func (a *App) GetAccounts() ([]database.Account, error) {
	return a.accountService.GetAccounts(a.ctx)
}

func (a *App) AddAccount(platform string, name string) (*database.Account, error) {
	return a.accountService.AddAccount(a.ctx, platform, name)
}

func (a *App) DeleteAccount(id int) error {
	return a.accountService.DeleteAccount(a.ctx, id)
}

func (a *App) UpdateAccount(account database.Account) error {
	return a.accountService.UpdateAccount(a.ctx, &account)
}

func (a *App) ValidateAccount(id int) (bool, error) {
	return a.accountService.ValidateAccount(a.ctx, id)
}

func (a *App) LoginAccount(id int) error {
	account, err := a.accountService.GetAccounts(a.ctx)
	if err != nil {
		a.emitEvent(config.EventLoginError, types.LoginErrorEvent{
			AccountID: id,
			Platform:  "",
			Error:     "failed to get accounts",
		})
		return fmt.Errorf("failed to get accounts: %w", err)
	}

	var targetAccount *database.Account
	for i := range account {
		if account[i].ID == id {
			targetAccount = &account[i]
			break
		}
	}

	if targetAccount == nil {
		a.emitEvent(config.EventLoginError, types.LoginErrorEvent{
			AccountID: id,
			Platform:  "",
			Error:     "account not found",
		})
		return fmt.Errorf("account not found")
	}

	err = a.accountService.LoginAccount(a.ctx, id)
	if err != nil {
		a.emitEvent(config.EventLoginError, types.LoginErrorEvent{
			AccountID: id,
			Platform:  targetAccount.Platform,
			Error:     err.Error(),
		})
		return err
	}

	a.emitEvent(config.EventLoginSuccess, types.LoginSuccessEvent{
		AccountID: id,
		Platform:  targetAccount.Platform,
		Username:  targetAccount.Username,
	})

	a.emitEvent(config.EventAccountStatusChanged, types.AccountStatusChangedEvent{
		AccountID: id,
		OldStatus: targetAccount.Status,
		NewStatus: config.AccountStatusValid,
	})

	// 登录成功后，重新注册上传器
	if a.scheduler != nil {
		a.registerUploaders()
	}

	return nil
}

func (a *App) ReloginAccount(id int) error {
	account, err := a.accountService.GetAccounts(a.ctx)
	if err != nil {
		a.emitEvent(config.EventLoginError, types.LoginErrorEvent{
			AccountID: id,
			Platform:  "",
			Error:     "failed to get accounts",
		})
		return fmt.Errorf("failed to get accounts: %w", err)
	}

	var targetAccount *database.Account
	for i := range account {
		if account[i].ID == id {
			targetAccount = &account[i]
			break
		}
	}

	if targetAccount == nil {
		a.emitEvent(config.EventLoginError, types.LoginErrorEvent{
			AccountID: id,
			Platform:  "",
			Error:     "account not found",
		})
		return fmt.Errorf("account not found")
	}

	err = a.accountService.ReloginAccount(a.ctx, id)
	if err != nil {
		a.emitEvent(config.EventLoginError, types.LoginErrorEvent{
			AccountID: id,
			Platform:  targetAccount.Platform,
			Error:     err.Error(),
		})
		return err
	}

	a.emitEvent(config.EventLoginSuccess, types.LoginSuccessEvent{
		AccountID: id,
		Platform:  targetAccount.Platform,
		Username:  targetAccount.Username,
	})

	a.emitEvent(config.EventAccountStatusChanged, types.AccountStatusChangedEvent{
		AccountID: id,
		OldStatus: targetAccount.Status,
		NewStatus: config.AccountStatusValid,
	})

	// 登录成功后，重新注册上传器
	if a.scheduler != nil {
		a.registerUploaders()
	}

	return nil
}

func (a *App) GetVideos() ([]database.Video, error) {
	return a.fileService.GetVideos(a.ctx)
}

func (a *App) AddVideo(filePath string) (*database.Video, error) {
	return a.fileService.AddVideo(a.ctx, filePath)
}

func (a *App) UpdateVideo(video database.Video) error {
	return a.fileService.UpdateVideo(a.ctx, &video)
}

func (a *App) DeleteVideo(id int) error {
	return a.fileService.DeleteVideo(a.ctx, id)
}

func (a *App) CreateUploadTask(
	videoID int,
	accountIDs []int,
	scheduleTime *string,
	metadata *string,
) ([]database.UploadTask, error) {
	// 解析元数据
	var taskMetadata *service.UploadTaskMetadata
	if metadata != nil && *metadata != "" {
		taskMetadata = &service.UploadTaskMetadata{}
		if err := json.Unmarshal([]byte(*metadata), taskMetadata); err != nil {
			return nil, fmt.Errorf("failed to parse metadata: %w", err)
		}
	}

	// 如果设置了定时时间，直接创建上传任务（立即执行，由平台处理定时发布）
	if scheduleTime != nil && *scheduleTime != "" {
		return a.uploadService.CreateUploadTask(a.ctx, videoID, accountIDs, scheduleTime, taskMetadata)
	}

	return a.uploadService.CreateUploadTask(a.ctx, videoID, accountIDs, nil, taskMetadata)
}

// createScheduledTasks 创建定时任务
func (a *App) createScheduledTasks(
	videoID int,
	accountIDs []int,
	scheduleTimeStr string,
	metadata *service.UploadTaskMetadata,
) ([]database.UploadTask, error) {
	// 获取视频信息
	videos, err := a.fileService.GetVideos(a.ctx)
	if err != nil {
		return nil, fmt.Errorf("获取视频列表失败: %w", err)
	}

	var video *database.Video
	for i := range videos {
		if videos[i].ID == videoID {
			video = &videos[i]
			break
		}
	}
	if video == nil {
		return nil, fmt.Errorf("视频不存在 (ID: %d)", videoID)
	}

	// 解析定时时间
	scheduleTime, err := time.Parse(time.RFC3339, scheduleTimeStr)
	if err != nil {
		// 尝试前端格式: 2006-01-02T15:04:05
		scheduleTime, err = time.Parse("2006-01-02T15:04:05", scheduleTimeStr)
		if err != nil {
			// 尝试其他格式: 2006-01-02 15:04
			scheduleTime, err = time.Parse("2006-01-02 15:04", scheduleTimeStr)
			if err != nil {
				return nil, fmt.Errorf("定时时间格式错误: %w", err)
			}
		}
	}

	// 验证时间范围（≥2小时且≤15天）
	now := time.Now()
	minTime := now.Add(2 * time.Hour)
	maxTime := now.Add(15 * 24 * time.Hour)

	if scheduleTime.Before(minTime) {
		return nil, fmt.Errorf("定时时间必须至少提前2小时")
	}
	if scheduleTime.After(maxTime) {
		return nil, fmt.Errorf("定时时间不能超过15天")
	}

	var tasks []database.UploadTask
	var errors []string

	for _, accountID := range accountIDs {
		account, err := a.accountService.GetAccountByID(a.ctx, accountID)
		if err != nil {
			errMsg := fmt.Sprintf("获取账号 %d 失败: %v", accountID, err)
			utils.Warn(fmt.Sprintf("[-] %s", errMsg))
			errors = append(errors, errMsg)
			continue
		}

		// 构建任务标题
		title := video.Title
		if metadata != nil && metadata.Common.Title != "" {
			title = metadata.Common.Title
		}

		// 应用平台特定字段
		platformFields := types.PlatformFields{}
		if metadata != nil && metadata.Platforms != nil {
			if pf, ok := metadata.Platforms[account.Platform]; ok {
				platformFields = pf
				if pf.Title != "" {
					title = pf.Title
				}
			}
		}

		// 创建定时任务
		scheduledTask := &database.ScheduledTask{
			ID:           fmt.Sprintf("task_%d_%d_%d", videoID, accountID, time.Now().Unix()),
			AccountID:    uint(accountID),
			Platform:     account.Platform,
			VideoPath:    video.FilePath,
			Title:        title,
			Description:  video.Description,
			Tags:         joinTags(video.Tags),
			ScheduleTime: scheduleTime,
			Status:       database.TaskStatusPending,
			Priority:     database.PriorityNormal,
			MaxRetries:   3,
		}

		// 添加到调度器
		if err := a.scheduler.AddTask(scheduledTask); err != nil {
			errMsg := fmt.Sprintf("添加定时任务到调度器失败 (账号: %s): %v", account.Name, err)
			utils.Warn(fmt.Sprintf("[-] %s", errMsg))
			errors = append(errors, errMsg)
			continue
		}

		// 同时创建 UploadTask 用于前端展示
		uploadTask := database.UploadTask{
			VideoID:      videoID,
			AccountID:    accountID,
			Platform:     account.Platform,
			Status:       config.TaskStatusPending,
			Progress:     0,
			ScheduleTime: &scheduleTimeStr,
			Title:        title,
		}
		// 应用平台特定字段
		if platformFields.Collection != "" {
			uploadTask.Collection = platformFields.Collection
		}
		if platformFields.Thumbnail != "" {
			uploadTask.Thumbnail = platformFields.Thumbnail
		}
		uploadTask.IsOriginal = platformFields.IsOriginal
		uploadTask.IsDraft = platformFields.IsDraft

		// 保存到数据库
		if err := database.DB.Create(&uploadTask).Error; err != nil {
			errMsg := fmt.Sprintf("创建上传任务记录失败 (账号: %s): %v", account.Name, err)
			utils.Warn(fmt.Sprintf("[-] %s", errMsg))
			errors = append(errors, errMsg)
			continue
		}

		tasks = append(tasks, uploadTask)
		utils.Info(fmt.Sprintf("[+] 定时任务已创建 - 视频: %s, 平台: %s, 时间: %s", video.Title, account.Platform, scheduleTimeStr))
	}

	// 如果没有成功创建任何任务，返回错误
	if len(tasks) == 0 {
		if len(errors) > 0 {
			return nil, fmt.Errorf("创建任务失败: %s", strings.Join(errors, "; "))
		}
		return nil, fmt.Errorf("没有成功创建任何任务")
	}

	// 如果部分成功，记录警告
	if len(errors) > 0 {
		utils.Warn(fmt.Sprintf("[-] 部分任务创建失败: %s", strings.Join(errors, "; ")))
	}

	return tasks, nil
}

// joinTags 将标签数组拼接为字符串
func joinTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	result := ""
	for i, tag := range tags {
		if i > 0 {
			result += ","
		}
		result += tag
	}
	return result
}

func (a *App) GetUploadTasks(status string) ([]database.UploadTask, error) {
	return a.uploadService.GetUploadTasks(a.ctx, status)
}

func (a *App) CancelUploadTask(id int) error {
	return a.uploadService.CancelUploadTask(a.ctx, id)
}

func (a *App) RetryUploadTask(id int) error {
	return a.uploadService.RetryUploadTask(a.ctx, id)
}

func (a *App) DeleteUploadTask(id int) error {
	return a.uploadService.DeleteUploadTask(a.ctx, id)
}

func (a *App) GetScheduleConfig() (*database.ScheduleConfig, error) {
	return a.scheduleService.GetScheduleConfig(a.ctx)
}

func (a *App) UpdateScheduleConfig(config database.ScheduleConfig) error {
	return a.scheduleService.UpdateScheduleConfig(a.ctx, &config)
}

func (a *App) GenerateScheduleTimes(videoCount int) ([]string, error) {
	times, err := a.scheduleService.GenerateScheduleTimes(a.ctx, videoCount)
	if err != nil {
		return nil, err
	}

	result := make([]string, len(times))
	for i, t := range times {
		result[i] = t.Format(time.RFC3339)
	}
	return result, nil
}

func (a *App) GetAppVersion() (types.AppVersion, error) {
	return types.AppVersion{
		Version:      config.AppVersion,
		BuildTime:    time.Now().Format("2006-01-02"),
		GoVersion:    runtime.Version(),
		WailsVersion: "v2.10.1",
	}, nil
}

func (a *App) SelectVideoFile() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("context not initialized")
	}

	selection, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择视频文件",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "视频文件 (*.mp4, *.mov, *.avi)", Pattern: "*.mp4;*.mov;*.avi"},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		},
	})

	if err != nil {
		return "", fmt.Errorf("open file dialog failed: %w", err)
	}

	return selection, nil
}

func (a *App) OpenDirectory(path string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open directory failed: %w", err)
	}

	return nil
}

// ============================================
// 平台特定 API（前端调用）
// ============================================

// GetCollections 获取平台合集列表（视频号）
func (a *App) GetCollections(platform string) ([]types.Collection, error) {
	// TODO: 实现获取合集列表逻辑
	// 这里需要根据平台调用相应的上传器获取合集
	utils.Info(fmt.Sprintf("[-] 获取 %s 合集列表", platform))

	// 返回模拟数据，实际实现需要从平台获取
	return []types.Collection{
		{Label: "默认合集", Value: "default"},
		{Label: "测试合集", Value: "test"},
	}, nil
}

// AutoSelectCover 自动选择推荐封面（从视频第一帧提取）
func (a *App) AutoSelectCover(videoID int) (*types.CoverInfo, error) {
	utils.Info(fmt.Sprintf("[-] 为视频 %d 自动选择封面", videoID))

	thumbnailPath, err := a.fileService.ExtractAndSaveThumbnail(a.ctx, videoID, 0)
	if err != nil {
		return nil, fmt.Errorf("提取封面失败: %w", err)
	}

	return &types.CoverInfo{
		ThumbnailPath: thumbnailPath,
	}, nil
}

// ExtractVideoFrame 从视频指定时间提取帧作为封面
func (a *App) ExtractVideoFrame(videoID int, timeSeconds int) (*types.CoverInfo, error) {
	utils.Info(fmt.Sprintf("[-] 从视频 %d 的 %d 秒处提取封面", videoID, timeSeconds))

	// 先获取视频信息
	video, err := a.fileService.GetVideoByID(a.ctx, videoID)
	if err != nil {
		utils.Error(fmt.Sprintf("[-] 获取视频信息失败: %v", err))
		return nil, fmt.Errorf("获取视频信息失败: %w", err)
	}

	utils.Info(fmt.Sprintf("[-] 视频路径: %s", video.FilePath))

	// 检查视频文件是否存在
	if _, err := os.Stat(video.FilePath); os.IsNotExist(err) {
		utils.Error(fmt.Sprintf("[-] 视频文件不存在: %s", video.FilePath))
		return nil, fmt.Errorf("视频文件不存在: %s", video.FilePath)
	}

	thumbnailPath, err := a.fileService.ExtractAndSaveThumbnail(a.ctx, videoID, timeSeconds)
	if err != nil {
		utils.Error(fmt.Sprintf("[-] 提取视频帧失败: %v", err))
		return nil, fmt.Errorf("提取视频帧失败: %w", err)
	}

	utils.Info(fmt.Sprintf("[-] 封面提取成功: %s", thumbnailPath))

	return &types.CoverInfo{
		ThumbnailPath: thumbnailPath,
	}, nil
}

// UploadThumbnail 上传本地图片作为封面
func (a *App) UploadThumbnail(videoID int, sourcePath string) (*types.CoverInfo, error) {
	utils.Info(fmt.Sprintf("[-] 为视频 %d 上传封面: %s", videoID, sourcePath))

	if sourcePath == "" {
		return nil, fmt.Errorf("封面路径不能为空")
	}

	thumbnailPath, err := a.fileService.SaveThumbnail(videoID, sourcePath)
	if err != nil {
		return nil, fmt.Errorf("保存封面失败: %w", err)
	}

	video, err := a.fileService.GetVideoByID(a.ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("获取视频信息失败: %w", err)
	}

	video.Thumbnail = thumbnailPath
	if err := a.fileService.UpdateVideo(a.ctx, video); err != nil {
		return nil, fmt.Errorf("更新视频封面失败: %w", err)
	}

	return &types.CoverInfo{
		ThumbnailPath: thumbnailPath,
	}, nil
}

// ClearThumbnail 清除视频封面
func (a *App) ClearThumbnail(videoID int) error {
	utils.Info(fmt.Sprintf("[-] 清除视频 %d 的封面", videoID))

	video, err := a.fileService.GetVideoByID(a.ctx, videoID)
	if err != nil {
		return fmt.Errorf("获取视频信息失败: %w", err)
	}

	if video.Thumbnail != "" {
		if err := os.Remove(video.Thumbnail); err != nil && !os.IsNotExist(err) {
			utils.Error(fmt.Sprintf("删除封面文件失败: %v", err))
		}
	}

	video.Thumbnail = ""
	if err := a.fileService.UpdateVideo(a.ctx, video); err != nil {
		return fmt.Errorf("清除视频封面失败: %w", err)
	}

	return nil
}

// ValidateProductLink 验证商品链接（抖音）
func (a *App) ValidateProductLink(link string) (*types.ProductLinkValidationResult, error) {
	utils.Info(fmt.Sprintf("[-] 验证商品链接: %s", link))

	// 简单的链接格式验证
	if link == "" {
		return &types.ProductLinkValidationResult{
			Valid: false,
			Error: "链接不能为空",
		}, nil
	}

	// 检查链接格式
	if !strings.HasPrefix(link, "http://") && !strings.HasPrefix(link, "https://") {
		return &types.ProductLinkValidationResult{
			Valid: false,
			Error: "链接格式不正确，必须以 http:// 或 https:// 开头",
		}, nil
	}

	// TODO: 实际实现需要调用抖音API验证链接有效性
	// 这里返回模拟成功结果
	return &types.ProductLinkValidationResult{
		Valid: true,
		Title: "商品标题（待获取）",
	}, nil
}

// SelectImageFile 选择图片文件
func (a *App) SelectImageFile() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("context not initialized")
	}

	selection, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择图片文件",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "图片文件 (*.jpg, *.jpeg, *.png, *.gif)", Pattern: "*.jpg;*.jpeg;*.png;*.gif"},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		},
	})

	if err != nil {
		return "", fmt.Errorf("open file dialog failed: %w", err)
	}

	return selection, nil
}

// SelectFile 选择文件
func (a *App) SelectFile(accept string) (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("context not initialized")
	}

	filters := []wailsRuntime.FileFilter{
		{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
	}

	// 根据 accept 参数设置过滤器
	if accept != "" && accept != "*/*" {
		filters = []wailsRuntime.FileFilter{
			{DisplayName: fmt.Sprintf("指定类型 (%s)", accept), Pattern: accept},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		}
	}

	selection, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title:   "选择文件",
		Filters: filters,
	})

	if err != nil {
		return "", fmt.Errorf("open file dialog failed: %w", err)
	}

	return selection, nil
}

// GetLogs 获取日志
func (a *App) GetLogs(query types.LogQuery) ([]types.SimpleLog, error) {
	return a.logService.Query(query), nil
}

// AddLog 添加日志（供内部使用）
func (a *App) AddLog(message string) {
	a.logService.Add(types.SimpleLog{
		Date:    time.Now().Format("2006/1/2"),
		Time:    time.Now().Format("15:04:05"),
		Message: message,
		Level:   types.LogLevelInfo,
	})
}

// SetLogDedupEnabled 设置日志归并开关
func (a *App) SetLogDedupEnabled(enabled bool) {
	a.logService.SetDedupEnabled(enabled)
	if enabled {
		utils.Info("[+] 日志归并已启用")
	} else {
		utils.Info("[-] 日志归并已禁用")
	}
}

// IsLogDedupEnabled 获取日志归并状态
func (a *App) IsLogDedupEnabled() bool {
	return a.logService.IsDedupEnabled()
}

// GetLogPlatforms 获取所有有日志的平台列表
func (a *App) GetLogPlatforms() []string {
	return a.logService.GetPlatforms()
}

// ============================================
// 截图管理 API
// ============================================

// GetScreenshotConfig 获取截图配置
func (a *App) GetScreenshotConfig() (*types.ScreenshotConfig, error) {
	return a.screenshotService.GetConfig(), nil
}

// UpdateScreenshotConfig 更新截图配置
func (a *App) UpdateScreenshotConfig(config types.ScreenshotConfig) error {
	return a.screenshotService.UpdateConfig(&config)
}

// GetScreenshots 获取截图列表
func (a *App) GetScreenshots(query types.ScreenshotQuery) (*types.ScreenshotListResult, error) {
	return a.screenshotService.ListScreenshots(query)
}

// DeleteScreenshot 删除单个截图
func (a *App) DeleteScreenshot(id string) error {
	return a.screenshotService.DeleteScreenshot(id)
}

// BatchDeleteScreenshots 批量删除截图
func (a *App) BatchDeleteScreenshots(ids []string) (int, error) {
	return a.screenshotService.BatchDeleteScreenshots(ids)
}

// DeleteAllScreenshots 删除所有截图
func (a *App) DeleteAllScreenshots() (int, error) {
	return a.screenshotService.DeleteAllScreenshots()
}

// GetPlatformScreenshotStats 获取各平台截图统计
func (a *App) GetPlatformScreenshotStats() ([]types.PlatformScreenshotConfig, error) {
	return a.screenshotService.GetPlatformScreenshotStats(), nil
}

// CleanOldScreenshots 清理旧截图
func (a *App) CleanOldScreenshots() (int, error) {
	return a.screenshotService.CleanOldScreenshots()
}

// OpenScreenshotDir 打开截图目录
func (a *App) OpenScreenshotDir(platform string) error {
	dir := a.screenshotService.GetScreenshotDir(platform)
	return a.OpenDirectory(dir)
}

// ============================================
// 浏览器无头模式配置 API
// ============================================

// GetHeadlessConfig 获取浏览器无头模式配置
func (a *App) GetHeadlessConfig() (bool, error) {
	return config.Config.Headless, nil
}

// SetHeadlessConfig 设置浏览器无头模式配置
func (a *App) SetHeadlessConfig(headless bool) error {
	config.Config.Headless = headless
	// 设置环境变量使配置持久化到新进程
	if headless {
		os.Setenv("FUPLOADER_HEADLESS", "true")
	} else {
		os.Setenv("FUPLOADER_HEADLESS", "false")
	}
	if headless {
		utils.Info("[+] 浏览器无头模式已启用")
	} else {
		utils.Info("[-] 浏览器无头模式已禁用")
	}
	return nil
}
