package baijiahao

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

func debugLog(format string, args ...interface{}) {
	if config.Config != nil && config.Config.DebugMode {
		utils.InfoWithPlatform("baijiahao", fmt.Sprintf("[调试] "+format, args...))
	}
}

var defaultPool *browser.Pool

func init() {
	defaultPool = browser.NewPoolFromConfig()
}

type Uploader struct {
	accountID   uint
	cookiePath  string
	platform    string
	browserPool *browser.Pool
	config      Config
}

func NewUploader(cookiePath string) *Uploader {
	u := &Uploader{
		accountID:   0,
		cookiePath:  cookiePath,
		platform:    "baijiahao",
		browserPool: nil,
		config:      DefaultConfig(),
	}
	debugLog("创建上传器 - 地址: %p, cookiePath: '%s'", u, cookiePath)
	if cookiePath == "" {
		utils.Warn("[Baijiahao] NewUploader 收到空的cookiePath!")
	}
	return u
}

func NewUploaderWithAccount(accountID uint) *Uploader {
	cookiePath := config.GetCookiePath("baijiahao", int(accountID))
	u := &Uploader{
		accountID:   accountID,
		cookiePath:  cookiePath,
		platform:    "baijiahao",
		browserPool: nil,
		config:      DefaultConfig(),
	}
	debugLog("创建上传器 - 地址: %p, accountID: %d, cookiePath: '%s'", u, accountID, cookiePath)
	return u
}

func NewUploaderWithPool(accountID uint, pool *browser.Pool) *Uploader {
	cookiePath := config.GetCookiePath("baijiahao", int(accountID))
	u := &Uploader{
		accountID:   accountID,
		cookiePath:  cookiePath,
		platform:    "baijiahao",
		browserPool: pool,
		config:      DefaultConfig(),
	}
	debugLog("创建上传器 - 地址: %p, accountID: %d, cookiePath: '%s'", u, accountID, cookiePath)
	return u
}

func NewUploaderWithConfig(accountID uint, pool *browser.Pool, cfg Config) *Uploader {
	cookiePath := config.GetCookiePath("baijiahao", int(accountID))
	u := &Uploader{
		accountID:   accountID,
		cookiePath:  cookiePath,
		platform:    "baijiahao",
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
	return defaultPool
}

func (u *Uploader) Upload(ctx context.Context, task *types.VideoTask) error {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("开始上传: %s", task.VideoPath))

	if _, err := os.Stat(task.VideoPath); err != nil {
		return fmt.Errorf("视频文件不存在: %w", err)
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
	if _, err := page.Goto("https://baijiahao.baidu.com/builder/rc/edit?type=videoV2", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		return fmt.Errorf("失败: 打开发布页面 - %w", err)
	}

	if err := page.Locator("div#formMain:visible").WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.PageLoadTimeout.Milliseconds())),
	}); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 等待页面加载 - 超时: %v", err))
	}
	time.Sleep(3 * time.Second)

	if err := u.uploadVideo(ctx, page, browserCtx, task.VideoPath); err != nil {
		return fmt.Errorf("失败: 上传视频 - %w", err)
	}

	if err := u.fillTitle(page, task.Title); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 填写标题 - %v", err))
	}

	if err := u.fillDescription(page, task.Description); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 填写描述 - %v", err))
	}

	if err := u.addTags(page, task.Tags); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 添加标签 - %v", err))
	}

	if err := u.selectCategory(page, task.Category); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 选择内容分类 - %v", err))
	}

	if err := u.setAICreation(page); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 勾选AI创作声明 - %v", err))
	}

	if err := u.setAutoAudio(page); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 勾选自动生成音频 - %v", err))
	}

	if task.Thumbnail != "" {
		if err := u.setCustomCover(page, task.Thumbnail); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置自定义封面 - %v", err))
		}
	}

	if task.ScheduleTime != nil && *task.ScheduleTime != "" {
		if err := u.setScheduleTime(page, *task.ScheduleTime); err != nil {
			return fmt.Errorf("失败: 设置定时发布 - %w", err)
		}
	} else {
		utils.InfoWithPlatform(u.platform, "准备发布...")
		if err := u.publish(page, browserCtx); err != nil {
			return fmt.Errorf("失败: 发布 - %w", err)
		}
	}

	utils.SuccessWithPlatform(u.platform, "发布成功")
	return nil
}

func (u *Uploader) fillTitle(page playwright.Page, title string) error {
	if title == "" {
		return nil
	}

	utils.InfoWithPlatform(u.platform, "填写标题...")

	if StringRuneLen(title) < u.config.TitleMinLength {
		title = title + " 你不知道的"
		utils.InfoWithPlatform(u.platform, fmt.Sprintf("标题少于%d字符，自动补全为: %s", u.config.TitleMinLength, title))
	}

	title = TruncateString(title, u.config.TitleMaxLength)

	titleInput := page.Locator(`div[contenteditable="true"][aria-placeholder="添加标题获得更多推荐"]`).First()
	if err := titleInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("失败: 填写标题 - 未找到输入框: %w", err)
	}

	if err := titleInput.Fill(title); err != nil {
		return fmt.Errorf("失败: 填写标题 - %w", err)
	}

	utils.InfoWithPlatform(u.platform, fmt.Sprintf("标题已填写: %s", title))
	time.Sleep(500 * time.Millisecond)
	return nil
}

func (u *Uploader) fillDescription(page playwright.Page, description string) error {
	if description == "" {
		return nil
	}

	utils.InfoWithPlatform(u.platform, "填写描述...")

	descInput := page.Locator(`textarea#desc`).First()
	if err := descInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("失败: 填写描述 - 未找到输入框: %w", err)
	}

	if err := descInput.Fill(description); err != nil {
		return fmt.Errorf("失败: 填写描述 - %w", err)
	}

	utils.InfoWithPlatform(u.platform, "描述已填写")
	time.Sleep(500 * time.Millisecond)
	return nil
}

func (u *Uploader) addTags(page playwright.Page, tags []string) error {
	if len(tags) == 0 {
		return nil
	}

	utils.InfoWithPlatform(u.platform, "添加标签...")

	tagInput := page.Locator(`input.cheetah-ui-pro-tag-input-container-tag-input`).First()
	if err := tagInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("失败: 添加标签 - 未找到输入框: %w", err)
	}

	for i, tag := range tags {
		cleanTag := strings.TrimSpace(tag)
		cleanTag = strings.ReplaceAll(cleanTag, "#", "")
		cleanTag = strings.ReplaceAll(cleanTag, " ", "")

		if cleanTag == "" {
			continue
		}

		if err := tagInput.Fill(cleanTag); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 添加标签 - 输入标签[%d]失败: %v", i, err))
			continue
		}
		time.Sleep(300 * time.Millisecond)

		if err := tagInput.Press("Enter"); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 添加标签 - 确认标签[%d]失败: %v", i, err))
			continue
		}
		time.Sleep(500 * time.Millisecond)
	}

	utils.InfoWithPlatform(u.platform, "标签添加完成")
	return nil
}

func (u *Uploader) selectCategory(page playwright.Page, category string) error {
	if category == "" {
		category = "科技"
	}

	utils.InfoWithPlatform(u.platform, fmt.Sprintf("选择内容分类: %s", category))

	categoryInput := page.Locator(`input#rc_select_22`).First()
	if err := categoryInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("失败: 选择内容分类 - 未找到选择器: %w", err)
	}

	if err := categoryInput.Click(); err != nil {
		return fmt.Errorf("失败: 选择内容分类 - 点击选择器失败: %w", err)
	}
	time.Sleep(1 * time.Second)

	categoryOption := page.Locator(fmt.Sprintf(`span:has-text("%s")`, category)).First()
	if err := categoryOption.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("失败: 选择内容分类 - 未找到选项: %w", err)
	}

	if err := categoryOption.Click(); err != nil {
		return fmt.Errorf("失败: 选择内容分类 - 点击选项失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, fmt.Sprintf("分类已选择: %s", category))
	time.Sleep(500 * time.Millisecond)
	return nil
}

func (u *Uploader) setAICreation(page playwright.Page) error {
	utils.InfoWithPlatform(u.platform, "勾选AI创作声明...")

	aiCheckbox := page.Locator(`span:has-text("AI创作声明")`).First()
	if err := aiCheckbox.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("失败: 勾选AI创作声明 - 未找到选项: %w", err)
	}

	if err := aiCheckbox.Click(); err != nil {
		return fmt.Errorf("失败: 勾选AI创作声明 - %w", err)
	}

	utils.InfoWithPlatform(u.platform, "AI创作声明已勾选")
	time.Sleep(500 * time.Millisecond)
	return nil
}

func (u *Uploader) setAutoAudio(page playwright.Page) error {
	utils.InfoWithPlatform(u.platform, "勾选自动生成音频...")

	audioCheckbox := page.Locator(`span:has-text("自动生成音频")`).First()
	if err := audioCheckbox.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("失败: 勾选自动生成音频 - 未找到选项: %w", err)
	}

	if err := audioCheckbox.Click(); err != nil {
		return fmt.Errorf("失败: 勾选自动生成音频 - %w", err)
	}

	utils.InfoWithPlatform(u.platform, "自动生成音频已勾选")
	time.Sleep(500 * time.Millisecond)
	return nil
}
