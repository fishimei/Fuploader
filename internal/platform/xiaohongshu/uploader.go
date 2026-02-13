package xiaohongshu

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"Fuploader/internal/config"
	"Fuploader/internal/platform/browser"
	"Fuploader/internal/types"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

type Uploader struct {
	accountID    uint
	cookiePath   string
	platform     string
	browserPool  *browser.Pool
	config       Config
	coverHandler *CoverHandler
}

var defaultPool *browser.Pool

func init() {
	defaultPool = browser.NewPoolFromConfig()
}

func NewUploader(cookiePath string) *Uploader {
	u := &Uploader{
		accountID:    0,
		cookiePath:   cookiePath,
		platform:     "xiaohongshu",
		browserPool:  defaultPool,
		config:       DefaultConfig(),
	}
	u.coverHandler = NewCoverHandler(u.config)
	debugLog("创建上传器 - 地址: %p, cookiePath: '%s'", u, cookiePath)
	if cookiePath == "" {
		utils.Warn("[XiaoHongShu] NewUploader 收到空的cookiePath!")
	}
	return u
}

func NewUploaderWithAccount(accountID uint) *Uploader {
	cookiePath := config.GetCookiePath("xiaohongshu", int(accountID))
	u := &Uploader{
		accountID:    accountID,
		cookiePath:   cookiePath,
		platform:     "xiaohongshu",
		browserPool:  defaultPool,
		config:       DefaultConfig(),
	}
	u.coverHandler = NewCoverHandler(u.config)
	debugLog("创建上传器 - 地址: %p, accountID: %d, cookiePath: '%s'", u, accountID, cookiePath)
	return u
}

func NewUploaderWithPool(accountID uint, pool *browser.Pool) *Uploader {
	cookiePath := config.GetCookiePath("xiaohongshu", int(accountID))
	u := &Uploader{
		accountID:    accountID,
		cookiePath:   cookiePath,
		platform:     "xiaohongshu",
		browserPool:  pool,
		config:       DefaultConfig(),
	}
	u.coverHandler = NewCoverHandler(u.config)
	debugLog("创建上传器 - 地址: %p, accountID: %d, cookiePath: '%s'", u, accountID, cookiePath)
	return u
}

func NewUploaderWithConfig(accountID uint, pool *browser.Pool, cfg Config) *Uploader {
	cookiePath := config.GetCookiePath("xiaohongshu", int(accountID))
	u := &Uploader{
		accountID:    accountID,
		cookiePath:   cookiePath,
		platform:     "xiaohongshu",
		browserPool:  pool,
		config:       cfg,
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
	return defaultPool
}

func (u *Uploader) Upload(ctx context.Context, task *types.VideoTask) error {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("开始上传: %s", task.VideoPath))

	if _, err := os.Stat(task.VideoPath); err != nil {
		return fmt.Errorf("失败: 开始上传 - 视频文件不存在: %w", err)
	}

	browserCtx, err := u.getBrowserPool().GetContextByAccount(ctx, u.accountID, u.cookiePath, nil)
	if err != nil {
		return fmt.Errorf("失败: 开始上传 - 获取浏览器失败: %w", err)
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		return fmt.Errorf("失败: 开始上传 - 获取页面失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "正在打开发布页面...")
	if _, err := page.Goto("https://creator.xiaohongshu.com/publish/publish?from=menu&target=video", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		return fmt.Errorf("失败: 开始上传 - 打开页面失败: %w", err)
	}
	time.Sleep(3 * time.Second)

	if err := u.uploadVideo(ctx, page, browserCtx, task.VideoPath); err != nil {
		return fmt.Errorf("失败: 上传视频 - %w", err)
	}

	time.Sleep(2 * time.Second)

	if err := u.fillTitle(page, task.Title); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 填写标题 - %v", err))
	}

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

	if task.Thumbnail != "" {
		if err := u.coverHandler.SetCover(page, task.Thumbnail); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置封面 - %v", err))
		}
	}

	if task.Location != "" {
		if err := u.setLocation(page, task.Location); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置位置 - %v", err))
		}
	}

	if task.ScheduleTime != nil && *task.ScheduleTime != "" {
		if err := u.setScheduleTime(page, *task.ScheduleTime); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置定时发布 - %v", err))
		}
	}

	utils.InfoWithPlatform(u.platform, "准备发布...")
	if err := u.publish(page, browserCtx, task.ScheduleTime != nil && *task.ScheduleTime != ""); err != nil {
		return fmt.Errorf("失败: 发布 - %w", err)
	}

	utils.SuccessWithPlatform(u.platform, "发布成功")
	return nil
}

func (u *Uploader) fillTitle(page playwright.Page, title string) error {
	if title == "" {
		return nil
	}

	utils.InfoWithPlatform(u.platform, "填写标题...")

	runes := []rune(title)
	if len(runes) > u.config.TitleMaxLength {
		title = string(runes[:u.config.TitleMaxLength])
	}

	newInput := page.Locator("input.d-text[placeholder*='标题']")
	newCount, _ := newInput.Count()
	if newCount > 0 {
		newInput.Fill(title)
	} else {
		oldInput := page.Locator("input.d-text[type='text']").First()
		oldCount, _ := oldInput.Count()
		if oldCount > 0 {
			oldInput.Click()
			page.Keyboard().Press("Backspace")
			page.Keyboard().Press("Control+KeyA")
			page.Keyboard().Press("Delete")
			page.Keyboard().Type(title)
			page.Keyboard().Press("Enter")
		} else {
			return fmt.Errorf("失败: 填写标题 - 未找到标题输入框")
		}
	}

	utils.InfoWithPlatform(u.platform, fmt.Sprintf("标题已填写: %s", title))
	time.Sleep(500 * time.Millisecond)
	return nil
}

func (u *Uploader) fillDescription(page playwright.Page, description string) error {
	utils.InfoWithPlatform(u.platform, "填写描述...")

	editor := page.Locator(".tiptap.ProseMirror")
	if err := editor.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("失败: 填写描述 - 未找到描述编辑器: %w", err)
	}

	if err := editor.Click(); err != nil {
		return fmt.Errorf("失败: 填写描述 - 点击编辑器失败: %w", err)
	}
	time.Sleep(300 * time.Millisecond)

	page.Keyboard().Press("Control+KeyA")
	page.Keyboard().Press("Delete")
	page.Keyboard().Type(description)

	utils.InfoWithPlatform(u.platform, "描述已填写")
	time.Sleep(500 * time.Millisecond)
	return nil
}

func (u *Uploader) addTags(page playwright.Page, tags []string) error {
	utils.InfoWithPlatform(u.platform, "添加标签...")

	cssSelector := ".tiptap.ProseMirror"
	editor := page.Locator(cssSelector)
	if err := editor.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("失败: 添加标签 - 未找到编辑器: %w", err)
	}

	if err := editor.Click(); err != nil {
		return fmt.Errorf("失败: 添加标签 - 点击编辑器失败: %w", err)
	}

	for _, tag := range tags {
		cleanTag := strings.TrimSpace(tag)
		cleanTag = strings.ReplaceAll(cleanTag, "#", "")
		if cleanTag == "" {
			continue
		}

		page.Type(cssSelector, "#"+cleanTag)
		page.Press(cssSelector, "Space")
		time.Sleep(500 * time.Millisecond)
	}

	utils.InfoWithPlatform(u.platform, "标签添加完成")
	return nil
}

func (u *Uploader) setLocation(page playwright.Page, location string) error {
	utils.InfoWithPlatform(u.platform, "设置位置...")

	locEle, err := page.WaitForSelector("div.d-text.d-select-placeholder.d-text-ellipsis.d-text-nowrap", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	})
	if err != nil {
		return fmt.Errorf("失败: 设置位置 - 未找到位置选择器: %w", err)
	}

	if err := locEle.Click(); err != nil {
		return fmt.Errorf("失败: 设置位置 - 点击位置选择器失败: %w", err)
	}
	time.Sleep(1 * time.Second)

	page.Keyboard().Type(location)
	time.Sleep(3 * time.Second)

	flexibleXPath := fmt.Sprintf(
		"//div[contains(@class, 'd-popover') and contains(@class, 'd-dropdown')]"+
			"//div[contains(@class, 'd-options-wrapper')]"+
			"//div[contains(@class, 'd-grid') and contains(@class, 'd-options')]"+
			"//div[contains(@class, 'name') and text()='%s']",
		location,
	)

	locationOption, err := page.WaitForSelector(flexibleXPath, playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	})
	if err == nil && locationOption != nil {
		_, _ = locationOption.Evaluate("element => element.scrollIntoViewIfNeeded()")

		isVisible, _ := locationOption.IsVisible()
		if isVisible {
			if err := locationOption.Click(); err != nil {
				return fmt.Errorf("失败: 设置位置 - 点击位置选项失败: %w", err)
			}
			return nil
		}
	}

	fallbackOption := page.Locator(fmt.Sprintf("div:has-text('%s')", location)).First()
	if count, _ := fallbackOption.Count(); count > 0 {
		fallbackOption.Click()
		return nil
	}

	return fmt.Errorf("失败: 设置位置 - 未找到位置选项: %s", location)
}
