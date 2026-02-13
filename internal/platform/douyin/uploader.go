package douyin

import (
	"context"
	"fmt"
	"time"

	"Fuploader/internal/config"
	"Fuploader/internal/platform/browser"
	"Fuploader/internal/types"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

func debugLog(format string, args ...interface{}) {
	if config.Config != nil && config.Config.DebugMode {
		utils.InfoWithPlatform("douyin", fmt.Sprintf("[调试] "+format, args...))
	}
}

type Uploader struct {
	accountID    uint
	cookiePath   string
	platform     string
	browserPool  *browser.Pool
	config       Config
	coverHandler *CoverHandler
}

func NewUploader(cookiePath string) *Uploader {
	u := &Uploader{
		accountID:   0,
		cookiePath:  cookiePath,
		platform:    "douyin",
		browserPool: browser.GetDefaultPool(),
		config:      DefaultConfig(),
	}
	u.coverHandler = NewCoverHandler(u.config)
	debugLog("创建上传器 - 地址: %p, cookiePath: '%s'", u, cookiePath)
	if cookiePath == "" {
		utils.WarnWithPlatform(u.platform, "失败: 创建上传器 - cookie路径为空")
	}
	return u
}

func NewUploaderWithAccount(accountID uint) *Uploader {
	cookiePath := config.GetCookiePath("douyin", int(accountID))
	u := &Uploader{
		accountID:   accountID,
		cookiePath:  cookiePath,
		platform:    "douyin",
		browserPool: browser.GetDefaultPool(),
		config:      DefaultConfig(),
	}
	u.coverHandler = NewCoverHandler(u.config)
	debugLog("创建上传器 - 地址: %p, accountID: %d, cookiePath: '%s'", u, accountID, cookiePath)
	return u
}

func NewUploaderWithPool(accountID uint, pool *browser.Pool) *Uploader {
	cookiePath := config.GetCookiePath("douyin", int(accountID))
	u := &Uploader{
		accountID:   accountID,
		cookiePath:  cookiePath,
		platform:    "douyin",
		browserPool: pool,
		config:      DefaultConfig(),
	}
	u.coverHandler = NewCoverHandler(u.config)
	debugLog("创建上传器 - 地址: %p, accountID: %d, cookiePath: '%s'", u, accountID, cookiePath)
	return u
}

func NewUploaderWithConfig(accountID uint, pool *browser.Pool, cfg Config) *Uploader {
	cookiePath := config.GetCookiePath("douyin", int(accountID))
	u := &Uploader{
		accountID:   accountID,
		cookiePath:  cookiePath,
		platform:    "douyin",
		browserPool: pool,
		config:      cfg,
	}
	u.coverHandler = NewCoverHandler(u.config)
	return u
}

func (u *Uploader) Platform() string {
	return u.platform
}

func (u *Uploader) getBrowserPool() *browser.Pool {
	if u.browserPool != nil {
		return u.browserPool
	}
	return browser.GetDefaultPool()
}

func (u *Uploader) Upload(ctx context.Context, task *types.VideoTask) error {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("开始上传: %s", task.VideoPath))

	if err := u.checkVideoExists(task.VideoPath); err != nil {
		return err
	}

	browserCtx, err := u.getBrowserPool().GetContextByAccount(ctx, u.accountID, u.cookiePath, nil)
	if err != nil {
		return fmt.Errorf("获取浏览器失败: %w", err)
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		return fmt.Errorf("获取页面失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "正在打开发布页面...")
	if _, err := page.Goto("https://creator.douyin.com/creator-micro/content/upload", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		return fmt.Errorf("失败: 打开发布页面 - %w", err)
	}
	time.Sleep(3 * time.Second)

	utils.InfoWithPlatform(u.platform, "正在上传视频...")
	if err := u.uploadVideo(ctx, page, browserCtx, task.VideoPath); err != nil {
		return fmt.Errorf("失败: 上传视频 - %w", err)
	}

	time.Sleep(2 * time.Second)

	if task.Title != "" {
		if err := u.fillTitle(page, task.Title); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 填写标题 - %v", err))
		}
	}

	if task.Description != "" || len(task.Tags) > 0 {
		if task.Description != "" {
			if err := u.fillDescription(page, task.Description); err != nil {
				utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 填写描述 - %v", err))
			}
		}
		if len(task.Tags) > 0 {
			if err := u.addTags(page, task.Tags); err != nil {
				utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 添加标签 - %v", err))
			}
		}
	}

	if err := u.coverHandler.SetCover(page, task.Thumbnail); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置封面 - %v", err))
	}

	if task.ProductLink != "" {
		if err := u.addProductLink(page, task.ProductLink, task.ProductTitle); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 添加商品链接 - %v", err))
		}
	}

	if task.SyncToutiao || task.SyncXigua {
		if err := u.setSyncOptions(page, task.SyncToutiao, task.SyncXigua); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置同步选项 - %v", err))
		}
	}

	if !task.AllowDownload {
		if err := u.setPermissions(page, task.AllowDownload); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置权限选项 - %v", err))
		}
	}

	if task.ScheduleTime != nil && *task.ScheduleTime != "" {
		if err := u.setScheduleTime(page, *task.ScheduleTime); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置定时发布 - %v", err))
		}
	}

	utils.InfoWithPlatform(u.platform, "准备发布...")
	if err := u.publish(page, browserCtx); err != nil {
		return fmt.Errorf("失败: 发布 - %w", err)
	}

	utils.SuccessWithPlatform(u.platform, "发布成功")
	return nil
}
