package tencent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"Fuploader/internal/config"
	"Fuploader/internal/platform/browser"
	"Fuploader/internal/types"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

func debugLog(format string, args ...interface{}) {
	if config.Config != nil && config.Config.DebugMode {
		utils.InfoWithPlatform("tencent", fmt.Sprintf("[调试] "+format, args...))
	}
}

type Uploader struct {
	accountID   uint
	cookiePath  string
	platform    string
	browserPool *browser.Pool
	config      Config
}

func NewUploader(cookiePath string) *Uploader {
	return NewUploaderWithPool(cookiePath, nil)
}

func NewUploaderWithCookiePath(cookiePath string) *Uploader {
	return NewUploaderWithPool(cookiePath, nil)
}

func NewUploaderWithPool(cookiePath string, pool *browser.Pool) *Uploader {
	u := &Uploader{
		accountID:   0,
		cookiePath:  cookiePath,
		platform:    "tencent",
		browserPool: pool,
		config:      DefaultConfig(),
	}
	debugLog("创建上传器 - 地址: %p, cookiePath: '%s'", u, cookiePath)
	return u
}

func NewUploaderWithAccount(accountID uint) *Uploader {
	cookiePath := config.GetCookiePath("tencent", int(accountID))
	u := &Uploader{
		accountID:   accountID,
		cookiePath:  cookiePath,
		platform:    "tencent",
		browserPool: browser.GetDefaultPool(),
		config:      DefaultConfig(),
	}
	debugLog("创建上传器 - 地址: %p, accountID: %d, cookiePath: '%s'", u, accountID, cookiePath)
	return u
}

func NewUploaderWithConfig(cookiePath string, pool *browser.Pool, cfg Config) *Uploader {
	u := &Uploader{
		accountID:   0,
		cookiePath:  cookiePath,
		platform:    "tencent",
		browserPool: pool,
		config:      cfg,
	}
	debugLog("创建上传器 - 地址: %p, cookiePath: '%s'", u, cookiePath)
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
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("开始上传: %s", filepath.Base(task.VideoPath)))

	if _, err := os.Stat(task.VideoPath); err != nil {
		return fmt.Errorf("失败: 检查视频文件 - 视频文件不存在: %w", err)
	}

	options := &browser.ContextOptions{
		EnableAntiDetect:  false,
		EnableRandomDelay: false,
		HumanLikeBehavior: false,
	}
	browserCtx, err := u.getBrowserPool().GetContextByAccount(ctx, u.accountID, u.cookiePath, options)
	if err != nil {
		return fmt.Errorf("失败: 获取浏览器上下文 - %w", err)
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		return fmt.Errorf("失败: 获取页面 - %w", err)
	}

	utils.InfoWithPlatform(u.platform, "正在打开发布页面...")
	if _, err := page.Goto("https://channels.weixin.qq.com/platform/post/create", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(30000),
	}); err != nil {
		return fmt.Errorf("失败: 打开发布页面 - %w", err)
	}

	if err := u.uploadVideo(ctx, page, browserCtx, task.VideoPath); err != nil {
		return fmt.Errorf("失败: 上传视频 - %w", err)
	}

	time.Sleep(2 * time.Second)

	if err := u.fillTitleAndDescription(page, task.Title, task.Description); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 填写标题 - %v", err))
	}

	if len(task.Tags) > 0 {
		if err := u.addTags(page, task.Tags); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 添加标签 - %v", err))
		}
	}

	if task.IsOriginal {
		if err := u.setOriginal(page, task.OriginalType); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置原创声明 - %v", err))
		}
	}

	if task.Collection != "" {
		if err := u.addToCollection(page, task.Collection); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 添加到合集 - %v", err))
		}
	}

	if task.Thumbnail != "" {
		if err := u.setCover(page, task.Thumbnail); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置封面 - %v", err))
		}
	}

	if err := u.setShortTitle(page, task.Title); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置短标题 - %v", err))
	}

	if task.ScheduleTime != nil && *task.ScheduleTime != "" {
		if err := u.setScheduleTime(page, *task.ScheduleTime); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置定时发布 - %v", err))
		}
	}

	if task.IsDraft {
		if err := u.saveDraft(page, browserCtx); err != nil {
			return fmt.Errorf("失败: 保存草稿 - %w", err)
		}
	} else {
		if err := u.publish(page, browserCtx); err != nil {
			return fmt.Errorf("失败: 发布 - %w", err)
		}
	}

	return nil
}

func (u *Uploader) Login() error {
	debugLog("Login开始 - cookiePath: '%s'", u.cookiePath)
	if u.cookiePath == "" {
		return fmt.Errorf("失败: 登录 - cookie路径为空")
	}

	ctx := context.Background()

	browserCtx, err := u.getBrowserPool().GetContextByAccount(ctx, u.accountID, u.cookiePath, nil)
	if err != nil {
		return fmt.Errorf("失败: 登录 - 获取浏览器上下文失败: %w", err)
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		return fmt.Errorf("失败: 登录 - 获取页面失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "正在打开发布页面...")
	if _, err := page.Goto("https://channels.weixin.qq.com/platform/post/create", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		return fmt.Errorf("失败: 登录 - 打开发布页面失败: %w", err)
	}

	time.Sleep(3 * time.Second)

	cookieConfig, ok := browser.GetCookieConfig("tencent")
	if !ok {
		return fmt.Errorf("失败: 登录 - 获取视频号Cookie配置失败")
	}

	if err := browserCtx.WaitForLoginCookies(cookieConfig); err != nil {
		return fmt.Errorf("失败: 登录 - 等待登录Cookie失败: %w", err)
	}

	utils.SuccessWithPlatform(u.platform, "登录成功")
	if err := browserCtx.SaveCookiesTo(u.cookiePath); err != nil {
		return fmt.Errorf("失败: 登录 - 保存Cookie失败: %w", err)
	}
	return nil
}
