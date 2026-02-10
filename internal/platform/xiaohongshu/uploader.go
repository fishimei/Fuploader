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

// debugLog 调试日志输出，仅在调试模式下显示
func debugLog(format string, args ...interface{}) {
	if config.Config != nil && config.Config.DebugMode {
		utils.InfoWithPlatform("xiaohongshu", fmt.Sprintf("[调试] "+format, args...))
	}
}

// browserPool 全局浏览器池实例
var browserPool *browser.Pool

func init() {
	browserPool = browser.NewPool(2, 5)
}

// Uploader 小红书上器
type Uploader struct {
	cookiePath string
	platform   string
}

// NewUploader 创建上传器
func NewUploader(cookiePath string) *Uploader {
	u := &Uploader{
		cookiePath: cookiePath,
		platform:   "xiaohongshu",
	}
	debugLog("创建上传器 - 地址: %p, cookiePath: '%s'", u, cookiePath)
	if cookiePath == "" {
		utils.Warn("[XiaoHongShu] NewUploader 收到空的cookiePath!")
	}
	return u
}

// Platform 返回平台名称
func (u *Uploader) Platform() string {
	return u.platform
}

// ValidateCookie 验证Cookie是否有效
func (u *Uploader) ValidateCookie(ctx context.Context) (bool, error) {
	utils.InfoWithPlatform(u.platform, "验证Cookie")

	if _, err := os.Stat(u.cookiePath); os.IsNotExist(err) {
		utils.WarnWithPlatform(u.platform, "Cookie文件不存在")
		return false, nil
	}

	browserCtx, err := browserPool.GetContext(ctx, u.cookiePath, nil)
	if err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("获取浏览器失败: %v", err))
		return false, nil
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("获取页面失败: %v", err))
		return false, nil
	}

	if _, err := page.Goto("https://creator.xiaohongshu.com/publish/publish?from=menu&target=video", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("打开页面失败: %v", err))
		return false, nil
	}

	time.Sleep(3 * time.Second)

	// 使用Cookie检测机制验证登录状态
	cookieConfig, ok := browser.GetCookieConfig("xiaohongshu")
	if !ok {
		return false, fmt.Errorf("获取小红书Cookie配置失败")
	}

	isValid, err := browserCtx.ValidateLoginCookies(cookieConfig)
	if err != nil {
		return false, fmt.Errorf("验证Cookie失败: %w", err)
	}

	if isValid {
		utils.SuccessWithPlatform(u.platform, "登录成功，所有必需Cookie已检测到")
	} else {
		utils.WarnWithPlatform(u.platform, "登录验证失败，缺少必需Cookie")
	}

	return isValid, nil
}

// Upload 上传视频
func (u *Uploader) Upload(ctx context.Context, task *types.VideoTask) error {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("开始上传: %s", task.VideoPath))

	if _, err := os.Stat(task.VideoPath); err != nil {
		return fmt.Errorf("视频文件不存在: %w", err)
	}

	browserCtx, err := browserPool.GetContext(ctx, u.cookiePath, nil)
	if err != nil {
		return fmt.Errorf("获取浏览器失败: %w", err)
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		return fmt.Errorf("获取页面失败: %w", err)
	}

	// 导航到上传页面
	utils.InfoWithPlatform(u.platform, "正在打开上传页面...")
	if _, err := page.Goto("https://creator.xiaohongshu.com/publish/publish?from=menu&target=video", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		return fmt.Errorf("打开上传页面失败: %w", err)
	}
	time.Sleep(3 * time.Second)

	// 上传视频
	if err := u.uploadVideo(ctx, page, browserCtx, task.VideoPath); err != nil {
		return fmt.Errorf("上传视频失败: %w", err)
	}

	time.Sleep(2 * time.Second)

	// 填写标题（限制30字符）
	if err := u.fillTitle(page, task.Title); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("填写标题失败: %v", err))
	}

	// 填写描述
	if task.Description != "" {
		if err := u.fillDescription(page, task.Description); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("填写描述失败: %v", err))
		}
	}

	// 添加话题标签
	if len(task.Tags) > 0 {
		if err := u.addTags(page, task.Tags); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("添加标签失败: %v", err))
		}
	}

	// 设置封面
	if task.Thumbnail != "" {
		if err := u.setCover(page, task.Thumbnail); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("设置封面失败: %v", err))
		}
	}

	// 设置位置
	if task.Location != "" {
		if err := u.setLocation(page, task.Location); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("设置位置失败: %v", err))
		}
	}

	// 设置定时发布
	if task.ScheduleTime != nil && *task.ScheduleTime != "" {
		if err := u.setScheduleTime(page, *task.ScheduleTime); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("设置定时发布失败: %v", err))
		}
	}

	// 点击发布
	utils.InfoWithPlatform(u.platform, "准备发布...")
	if err := u.publish(page, browserCtx, task.ScheduleTime != nil && *task.ScheduleTime != ""); err != nil {
		return fmt.Errorf("发布失败: %w", err)
	}

	utils.SuccessWithPlatform(u.platform, "发布成功")
	return nil
}

// uploadVideo 上传视频
func (u *Uploader) uploadVideo(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext, videoPath string) error {
	utils.InfoWithPlatform(u.platform, "正在上传视频...")

	// 定位文件输入框
	input := page.Locator("div[class^='upload-content'] input[class='upload-input']")
	if err := input.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(10000),
	}); err != nil {
		// 兜底：尝试通用选择器
		input = page.Locator("input[type='file']").First()
		if err := input.WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(5000),
		}); err != nil {
			return fmt.Errorf("未找到文件输入框: %w", err)
		}
	}

	if err := input.SetInputFiles(videoPath); err != nil {
		return fmt.Errorf("设置视频文件失败: %w", err)
	}

	// 等待上传完成
	utils.InfoWithPlatform(u.platform, "等待视频上传完成...")
	if err := u.waitForUploadComplete(ctx, page, browserCtx); err != nil {
		return err
	}

	return nil
}

// waitForUploadComplete 等待视频上传完成
func (u *Uploader) waitForUploadComplete(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext) error {
	uploadTimeout := 5 * time.Minute
	uploadStartTime := time.Now()

	for time.Since(uploadStartTime) < uploadTimeout {
		select {
		case <-ctx.Done():
			return fmt.Errorf("上传已取消")
		default:
		}

		if browserCtx.IsPageClosed() {
			return fmt.Errorf("浏览器已关闭")
		}

		// 检测方式1：明确等待"上传成功"文本
		uploadSuccess, err := u.detectUploadSuccess(page)
		if err == nil && uploadSuccess {
			utils.InfoWithPlatform(u.platform, "视频上传成功")
			return nil
		}

		// 检测方式2：检查"重新上传"按钮出现
		reuploadCount, _ := page.Locator("[class^=\"long-card\"] div:has-text(\"重新上传\")").Count()
		if reuploadCount > 0 {
			utils.InfoWithPlatform(u.platform, "视频上传完成（检测到重新上传按钮）")
			return nil
		}

		// 检测方式3：检查视频预览区域
		videoPreview := page.Locator("video, .video-preview, [class*='preview']").First()
		if count, _ := videoPreview.Count(); count > 0 {
			if visible, _ := videoPreview.IsVisible(); visible {
				utils.InfoWithPlatform(u.platform, "视频上传完成（检测到预览）")
				return nil
			}
		}

		// 检测上传失败
		errorCount, _ := page.Locator("div.progress-div > div:has-text(\"上传失败\")").Count()
		if errorCount > 0 {
			utils.WarnWithPlatform(u.platform, "检测到上传失败，尝试重试...")
			retryInput := page.Locator("div.progress-div [class^=\"upload-btn-input\"]")
			if err := retryInput.SetInputFiles(page.URL()); err != nil {
				retryInput = page.Locator("input[type='file']")
				retryInput.SetInputFiles(page.URL())
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("上传超时")
}

// detectUploadSuccess 检测上传是否成功
func (u *Uploader) detectUploadSuccess(page playwright.Page) (bool, error) {
	// 使用 WaitForSelector 等待元素出现（与Python的 wait_for_selector 一致）
	uploadInput, err := page.WaitForSelector("input.upload-input", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(3000),
	})
	if err != nil {
		return false, err
	}

	// 使用 QuerySelector 查找兄弟元素（与Python的 query_selector 一致）
	previewNew, err := uploadInput.QuerySelector("xpath=following-sibling::div[contains(@class, 'preview-new')]")
	if err != nil || previewNew == nil {
		return false, fmt.Errorf("未找到预览区域")
	}

	// 使用 QuerySelectorAll 获取所有stage元素（与Python的 query_selector_all 一致）
	stageElements, err := previewNew.QuerySelectorAll("div.stage")
	if err != nil || len(stageElements) == 0 {
		return false, fmt.Errorf("未找到stage元素")
	}

	// 遍历检查文本内容（与Python的 evaluate 一致）
	for _, stage := range stageElements {
		textContent, err := stage.TextContent()
		if err != nil {
			continue
		}
		if strings.Contains(textContent, "上传成功") {
			return true, nil
		}
	}

	return false, fmt.Errorf("未检测到上传成功")
}

// fillTitle 填写标题（限制30字符）
func (u *Uploader) fillTitle(page playwright.Page, title string) error {
	if title == "" {
		return nil
	}

	utils.InfoWithPlatform(u.platform, "填写标题...")

	// 限制30字符
	if len(title) > 30 {
		runes := []rune(title)
		if len(runes) > 30 {
			title = string(runes[:30])
		}
	}

	// 尝试新版输入框（根据截图使用更稳定的选择器）
	newInput := page.Locator("input.d-text[placeholder*='标题']")
	newCount, _ := newInput.Count()
	if newCount > 0 {
		// 新版直接 fill
		newInput.Fill(title)
	} else {
		// 兜底：尝试通用选择器
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
			return fmt.Errorf("未找到标题输入框")
		}
	}

	utils.InfoWithPlatform(u.platform, fmt.Sprintf("标题已填写: %s", title))
	time.Sleep(500 * time.Millisecond)
	return nil
}

// fillDescription 填写描述
func (u *Uploader) fillDescription(page playwright.Page, description string) error {
	utils.InfoWithPlatform(u.platform, "填写描述...")

	// 定位富文本编辑器（根据截图使用 .tiptap.ProseMirror）
	editor := page.Locator(".tiptap.ProseMirror")
	if err := editor.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("未找到描述编辑器: %w", err)
	}

	if err := editor.Click(); err != nil {
		return fmt.Errorf("点击编辑器失败: %w", err)
	}
	time.Sleep(300 * time.Millisecond)

	// 清空并输入
	page.Keyboard().Press("Control+KeyA")
	page.Keyboard().Press("Delete")
	page.Keyboard().Type(description)

	utils.InfoWithPlatform(u.platform, "描述已填写")
	time.Sleep(500 * time.Millisecond)
	return nil
}

// addTags 添加话题标签
func (u *Uploader) addTags(page playwright.Page, tags []string) error {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("添加%d个标签...", len(tags)))

	// 定位编辑器（根据截图使用 .tiptap.ProseMirror）
	cssSelector := ".tiptap.ProseMirror"
	editor := page.Locator(cssSelector)
	if err := editor.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(3000),
	}); err != nil {
		return fmt.Errorf("未找到编辑器: %w", err)
	}

	if err := editor.Click(); err != nil {
		return fmt.Errorf("点击编辑器失败: %w", err)
	}

	// 在指定选择器上输入
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

// setCover 设置封面
func (u *Uploader) setCover(page playwright.Page, coverPath string) error {
	if _, err := os.Stat(coverPath); err != nil {
		return fmt.Errorf("封面文件不存在: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "设置封面...")

	// 点击封面设置按钮
	coverBtn := page.GetByText("选择封面").First()
	if err := coverBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("未找到封面设置按钮: %w", err)
	}

	if err := coverBtn.Click(); err != nil {
		return fmt.Errorf("点击封面设置按钮失败: %w", err)
	}
	time.Sleep(2 * time.Second)

	// 点击"设置竖封面"（如果有）
	verticalCoverBtn := page.GetByText("设置竖封面").First()
	if count, _ := verticalCoverBtn.Count(); count > 0 {
		verticalCoverBtn.Click()
		time.Sleep(2 * time.Second)
	}

	// 查找文件输入框
	coverInput := page.Locator("div[class^='semi-upload upload'] >> input.semi-upload-hidden-input").First()
	if err := coverInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		// 兜底：尝试通用选择器
		coverInput = page.Locator("input[type='file']").First()
		if err := coverInput.WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(3000),
		}); err != nil {
			return fmt.Errorf("未找到封面文件输入框: %w", err)
		}
	}

	if err := coverInput.SetInputFiles(coverPath); err != nil {
		return fmt.Errorf("上传封面失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "封面上传中...")
	time.Sleep(2 * time.Second)

	// 点击完成按钮
	finishBtn := page.Locator("div[class^='extractFooter'] button:visible:has-text('完成')").First()
	if count, _ := finishBtn.Count(); count > 0 {
		if err := finishBtn.Click(); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("点击完成按钮失败: %v", err))
		}
	}
	time.Sleep(2 * time.Second)

	utils.InfoWithPlatform(u.platform, "封面设置完成")
	return nil
}

// setLocation 设置位置
func (u *Uploader) setLocation(page playwright.Page, location string) error {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("设置位置: %s", location))

	// 等待并获取位置选择器（与Python的 wait_for_selector 一致）
	locEle, err := page.WaitForSelector("div.d-text.d-select-placeholder.d-text-ellipsis.d-text-nowrap", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(3000),
	})
	if err != nil {
		return fmt.Errorf("未找到位置选择器: %w", err)
	}

	// 点击位置选择器
	if err := locEle.Click(); err != nil {
		return fmt.Errorf("点击位置选择器失败: %w", err)
	}
	time.Sleep(1 * time.Second)

	// 输入位置
	page.Keyboard().Type(location)
	time.Sleep(3 * time.Second)

	// 选择匹配的位置选项（与Python的灵活XPath一致）
	flexibleXPath := fmt.Sprintf(
		"//div[contains(@class, 'd-popover') and contains(@class, 'd-dropdown')]"+
			"//div[contains(@class, 'd-options-wrapper')]"+
			"//div[contains(@class, 'd-grid') and contains(@class, 'd-options')]"+
			"//div[contains(@class, 'name') and text()='%s']",
		location,
	)

	// 使用 WaitForSelector 等待选项出现（与Python一致）
	locationOption, err := page.WaitForSelector(flexibleXPath, playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(3000),
	})
	if err == nil && locationOption != nil {
		// 滚动到元素可见（与Python的 scroll_into_view_if_needed 一致）
		_, _ = locationOption.Evaluate("element => element.scrollIntoViewIfNeeded()")

		// 检查可见性（与Python的 is_visible 一致）
		isVisible, _ := locationOption.IsVisible()
		if isVisible {
			if err := locationOption.Click(); err != nil {
				return fmt.Errorf("点击位置选项失败: %w", err)
			}
			utils.InfoWithPlatform(u.platform, "位置设置完成")
			return nil
		}
	}

	// 兜底：尝试模糊匹配
	fallbackOption := page.Locator(fmt.Sprintf("div:has-text('%s')", location)).First()
	if count, _ := fallbackOption.Count(); count > 0 {
		fallbackOption.Click()
		utils.InfoWithPlatform(u.platform, "位置设置完成（使用模糊匹配）")
		return nil
	}

	return fmt.Errorf("未找到位置选项: %s", location)
}

// setScheduleTime 设置定时发布
func (u *Uploader) setScheduleTime(page playwright.Page, scheduleTime string) error {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("设置定时发布时间: %s", scheduleTime))

	// 解析时间
	targetTime, err := time.Parse("2006-01-02 15:04", scheduleTime)
	if err != nil {
		return fmt.Errorf("解析时间失败: %w", err)
	}

	// 点击定时发布选项
	labelElement := page.Locator("label:has-text('定时发布')")
	if err := labelElement.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(3000),
	}); err != nil {
		return fmt.Errorf("未找到定时发布选项: %w", err)
	}

	if err := labelElement.Click(); err != nil {
		return fmt.Errorf("点击定时发布失败: %w", err)
	}
	time.Sleep(1 * time.Second)

	// 选择时间
	scheduleInput := page.Locator(".el-input__inner[placeholder=\"选择日期和时间\"]")
	if err := scheduleInput.Click(); err != nil {
		return fmt.Errorf("点击时间输入框失败: %w", err)
	}
	time.Sleep(500 * time.Millisecond)

	// 输入时间
	timeStr := targetTime.Format("2006-01-02 15:04")
	page.Keyboard().Press("Control+KeyA")
	page.Keyboard().Type(timeStr)
	page.Keyboard().Press("Enter")

	utils.InfoWithPlatform(u.platform, fmt.Sprintf("定时发布时间设置完成: %s", timeStr))
	time.Sleep(1 * time.Second)
	return nil
}

// publish 点击发布并检测结果
func (u *Uploader) publish(page playwright.Page, browserCtx *browser.PooledContext, isScheduled bool) error {
	// 检测发布结果（与Python的 while True 一致）
	publishTimeout := 30 * time.Second
	publishStart := time.Now()
	var waitErr error

	for time.Since(publishStart) < publishTimeout {
		if browserCtx.IsPageClosed() {
			return fmt.Errorf("浏览器已关闭")
		}

		// 每次循环重新定位并点击发布按钮（与Python一致）
		if isScheduled {
			button := page.Locator("button:has-text('定时发布')")
			if err := button.Click(); err != nil {
				utils.WarnWithPlatform(u.platform, fmt.Sprintf("点击定时发布按钮失败: %v", err))
			}
		} else {
			button := page.Locator("button:has-text('发布')")
			if err := button.Click(); err != nil {
				utils.WarnWithPlatform(u.platform, fmt.Sprintf("点击发布按钮失败: %v", err))
			}
		}

		// 等待页面跳转（与Python的 wait_for_url 一致，使用完整URL）
		waitErr = page.WaitForURL("https://creator.xiaohongshu.com/publish/success?**", playwright.PageWaitForURLOptions{
			Timeout: playwright.Float(3000),
		})
		if waitErr == nil {
			utils.InfoWithPlatform(u.platform, "视频发布成功")
			return nil
		}

		// 截图（与Python的 screenshot 一致）
		_, _ = page.Screenshot(playwright.PageScreenshotOptions{
			FullPage: playwright.Bool(true),
		})
		utils.InfoWithPlatform(u.platform, "视频正在发布中...")

		// 等待0.5秒继续（与Python的 sleep(0.5) 一致）
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("发布超时")
}

// Login 登录
func (u *Uploader) Login() error {
	debugLog("Login开始 - cookiePath: '%s'", u.cookiePath)
	if u.cookiePath == "" {
		return fmt.Errorf("cookie路径为空")
	}

	ctx := context.Background()
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("Cookie保存路径: %s", u.cookiePath))

	browserCtx, err := browserPool.GetContext(ctx, "", nil)
	if err != nil {
		return fmt.Errorf("获取浏览器失败: %w", err)
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		return fmt.Errorf("获取页面失败: %w", err)
	}

	// 先访问主页模拟正常用户行为
	utils.InfoWithPlatform(u.platform, "正在访问主页...")
	if _, err := page.Goto("https://www.xiaohongshu.com", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		utils.Warn(fmt.Sprintf("[-] 访问主页失败，直接访问登录页: %v", err))
	} else {
		utils.InfoWithPlatform(u.platform, "模拟浏览主页...")
		_, _ = page.Evaluate("window.scrollBy(0, 200)")
		time.Sleep(2 * time.Second)
	}

	utils.InfoWithPlatform(u.platform, "正在打开创作者中心登录页...")
	if _, err := page.Goto("https://creator.xiaohongshu.com/login", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		return fmt.Errorf("打开登录页面失败: %w", err)
	}

	time.Sleep(3 * time.Second)

	utils.InfoWithPlatform(u.platform, "请在浏览器窗口中完成登录...")

	// 使用Cookie检测机制等待登录成功
	cookieConfig, ok := browser.GetCookieConfig("xiaohongshu")
	if !ok {
		return fmt.Errorf("获取小红书Cookie配置失败")
	}

	if err := browserCtx.WaitForLoginCookies(cookieConfig); err != nil {
		return fmt.Errorf("等待登录Cookie失败: %w", err)
	}

	utils.SuccessWithPlatform(u.platform, "登录成功，所有必需Cookie已获取")
	if err := browserCtx.SaveCookiesTo(u.cookiePath); err != nil {
		return fmt.Errorf("保存Cookie失败: %w", err)
	}
	utils.InfoWithPlatform(u.platform, "Cookie已保存")
	return nil
}
