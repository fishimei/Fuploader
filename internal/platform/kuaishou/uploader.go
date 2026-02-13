package kuaishou

import (
	"context"
	"fmt"
	"os"
	"time"

	"Fuploader/internal/config"
	"Fuploader/internal/platform/browser"
	"Fuploader/internal/types"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

type Uploader struct {
	accountID   uint
	cookiePath  string
	platform    string
	browserPool *browser.Pool
	config      Config
}

func NewUploader(cookiePath string) *Uploader {
	return NewUploaderWithPool(0, cookiePath, nil)
}

func NewUploaderWithAccount(accountID uint) *Uploader {
	cookiePath := config.GetCookiePath("kuaishou", int(accountID))
	return NewUploaderWithPool(accountID, cookiePath, nil)
}

func NewUploaderWithPool(accountID uint, cookiePath string, pool *browser.Pool) *Uploader {
	u := &Uploader{
		accountID:   accountID,
		cookiePath:  cookiePath,
		platform:    "kuaishou",
		browserPool: pool,
		config:      DefaultConfig(),
	}
	debugLog("创建上传器 - 地址: %p, accountID: %d, cookiePath: '%s'", u, accountID, cookiePath)
	return u
}

func NewUploaderWithConfig(accountID uint, cookiePath string, pool *browser.Pool, cfg Config) *Uploader {
	u := &Uploader{
		accountID:   accountID,
		cookiePath:  cookiePath,
		platform:    "kuaishou",
		browserPool: pool,
		config:      cfg,
	}
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

	if _, err := os.Stat(task.VideoPath); err != nil {
		return fmt.Errorf("视频文件不存在: %v", err)
	}

	browserCtx, err := u.getBrowserPool().GetContextByAccount(ctx, u.accountID, u.cookiePath, nil)
	if err != nil {
		return fmt.Errorf("获取浏览器失败: %v", err)
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		return fmt.Errorf("获取页面失败: %v", err)
	}

	utils.InfoWithPlatform(u.platform, "正在打开发布页面...")
	if _, err := page.Goto("https://cp.kuaishou.com/article/publish/video", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		return fmt.Errorf("打开发布页面失败: %v", err)
	}
	time.Sleep(3 * time.Second)

	if err := u.handleNewFeatureGuide(page); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("处理新功能引导失败: %v", err))
	}

	if err := u.uploadVideo(ctx, page, browserCtx, task.VideoPath); err != nil {
		return fmt.Errorf("上传视频失败: %v", err)
	}

	time.Sleep(2 * time.Second)

	if err := u.handleSkipPopup(page); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("关闭弹窗失败: %v", err))
	}

	allowDownload := false
	if task.AllowDownload {
		allowDownload = true
	}
	if err := u.setDownloadPermission(page, allowDownload); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("设置下载权限失败: %v", err))
	}

	if task.Thumbnail != "" {
		if err := u.setCover(page, task.Thumbnail); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("设置封面失败: %v", err))
		}
	}

	if err := u.fillDescription(page, task.Title, task.Description); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("填写描述失败: %v", err))
	}

	if len(task.Tags) > 0 {
		if err := u.addTags(page, task.Tags); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("添加标签失败: %v", err))
		}
	}

	if task.ScheduleTime != nil && *task.ScheduleTime != "" {
		if err := u.setScheduleTime(page, *task.ScheduleTime); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("设置定时发布失败: %v", err))
		}
	}

	utils.InfoWithPlatform(u.platform, "准备发布...")
	if err := u.publish(page, browserCtx); err != nil {
		return fmt.Errorf("发布失败: %v", err)
	}

	utils.SuccessWithPlatform(u.platform, "发布成功")
	return nil
}
