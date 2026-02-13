package tiktok

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

func debugLog(format string, args ...interface{}) {
	if config.Config != nil && config.Config.DebugMode {
		utils.InfoWithPlatform("tiktok", fmt.Sprintf("[调试] "+format, args...))
	}
}

type Uploader struct {
	accountID   uint
	cookiePath  string
	platform    string
	browserPool *browser.Pool
}

func NewUploader(cookiePath string) *Uploader {
	u := &Uploader{
		accountID:   0,
		cookiePath:  cookiePath,
		platform:    "tiktok",
		browserPool: browser.GetDefaultPool(),
	}
	debugLog("创建上传器 - 地址: %p, cookiePath: '%s'", u, cookiePath)
	if cookiePath == "" {
		utils.Warn("[TikTok] NewUploader 收到空的cookiePath!")
	}
	return u
}

func NewUploaderWithAccount(accountID uint) *Uploader {
	cookiePath := config.GetCookiePath("tiktok", int(accountID))
	u := &Uploader{
		accountID:   accountID,
		cookiePath:  cookiePath,
		platform:    "tiktok",
		browserPool: browser.GetDefaultPool(),
	}
	debugLog("创建上传器 - 地址: %p, accountID: %d, cookiePath: '%s'", u, accountID, cookiePath)
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
		return fmt.Errorf("视频文件不存在: %w", err)
	}

	pool := u.getBrowserPool()
	browserCtx, err := pool.GetContextByAccount(ctx, u.accountID, u.cookiePath, u.getContextOptions())
	if err != nil {
		return fmt.Errorf("获取浏览器失败: %w", err)
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		return fmt.Errorf("获取页面失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "正在打开发布页面...")
	if _, err := page.Goto("https://www.tiktok.com/tiktokstudio/upload", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		return fmt.Errorf("打开发布页面失败: %w", err)
	}
	time.Sleep(3 * time.Second)

	locators := GetLocators()
	var locatorBase playwright.Locator
	iframeCount, _ := page.Locator(locators.Iframe).Count()
	if iframeCount > 0 {
		frame := page.FrameLocator(locators.Iframe)
		locatorBase = frame.Locator(locators.Container)
		debugLog("检测到iframe结构")
	} else {
		locatorBase = page.Locator(locators.Container)
		debugLog("使用普通容器结构")
	}

	if err := u.uploadVideo(ctx, page, browserCtx, locatorBase, task.VideoPath); err != nil {
		return fmt.Errorf("上传视频失败: %w", err)
	}

	time.Sleep(2 * time.Second)

	if err := u.fillTitleAndDescription(locatorBase, task.Title, task.Description); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("填写标题和描述失败: %v", err))
	}

	if len(task.Tags) > 0 {
		if err := u.addTags(locatorBase, task.Tags); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("添加标签失败: %v", err))
		}
	}

	if task.Thumbnail != "" {
		if err := u.setCover(page, locatorBase, task.Thumbnail); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("设置封面失败: %v", err))
		}
	}

	if task.ScheduleTime != nil && *task.ScheduleTime != "" {
		if err := u.setScheduleTime(locatorBase, *task.ScheduleTime); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("设置定时发布失败: %v", err))
		}
	}

	utils.InfoWithPlatform(u.platform, "准备发布...")
	if err := u.publish(ctx, page, locatorBase, browserCtx); err != nil {
		return fmt.Errorf("发布失败: %w", err)
	}

	utils.SuccessWithPlatform(u.platform, "发布成功")
	return nil
}

func (u *Uploader) getContextOptions() *browser.ContextOptions {
	return &browser.ContextOptions{
		UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:120.0) Gecko/20100101 Firefox/120.0",
		Viewport:    &playwright.Size{Width: 1920, Height: 1080},
		Locale:      "en-GB",
		TimezoneId:  "Europe/London",
		Geolocation: &playwright.Geolocation{Latitude: 51.5074, Longitude: -0.1278},
		ExtraHeaders: map[string]string{
			"Accept-Language": "en-GB,en;q=0.9",
		},
	}
}
