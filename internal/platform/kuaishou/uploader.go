package kuaishou

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
		utils.InfoWithPlatform("kuaishou", fmt.Sprintf("[调试] "+format, args...))
	}
}

// browserPool 全局浏览器池实例
var browserPool *browser.Pool

func init() {
	browserPool = browser.NewPool(2, 5)
}

// Uploader 快手上传器
type Uploader struct {
	cookiePath string
	platform   string
}

// NewUploader 创建上传器
func NewUploader(cookiePath string) *Uploader {
	u := &Uploader{
		cookiePath: cookiePath,
		platform:   "kuaishou",
	}
	debugLog("创建上传器 - 地址: %p, cookiePath: '%s'", u, cookiePath)
	if cookiePath == "" {
		utils.Warn("[Kuaishou] NewUploader 收到空的cookiePath!")
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

	if _, err := page.Goto("https://cp.kuaishou.com/", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("打开页面失败: %v", err))
		return false, nil
	}

	time.Sleep(3 * time.Second)

	// 使用Cookie检测机制验证登录状态
	cookieConfig, ok := browser.GetCookieConfig("kuaishou")
	if !ok {
		return false, fmt.Errorf("获取快手Cookie配置失败")
	}

	isValid, err := browserCtx.ValidateLoginCookies(cookieConfig)
	if err != nil {
		return false, fmt.Errorf("验证Cookie失败: %w", err)
	}

	if isValid {
		utils.InfoWithPlatform(u.platform, "检测到kuaishou.server.web_ph Cookie，验证通过")
	} else {
		utils.InfoWithPlatform(u.platform, "未检测到kuaishou.server.web_ph Cookie，验证失败")
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
	if _, err := page.Goto("https://cp.kuaishou.com/article/publish/video", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		return fmt.Errorf("打开上传页面失败: %w", err)
	}
	time.Sleep(3 * time.Second)

	// 处理新功能引导
	if err := u.handleNewFeatureGuide(page); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("处理新功能引导失败: %v", err))
	}

	// 上传视频
	if err := u.uploadVideo(ctx, page, browserCtx, task.VideoPath); err != nil {
		return fmt.Errorf("上传视频失败: %w", err)
	}

	time.Sleep(2 * time.Second)

	// 填写描述和标题
	if err := u.fillDescription(page, task.Title, task.Description); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("填写描述失败: %v", err))
	}

	// 添加话题标签（最多3个）
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

	// 设置定时发布
	if task.ScheduleTime != nil && *task.ScheduleTime != "" {
		if err := u.setScheduleTime(page, *task.ScheduleTime); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("设置定时发布失败: %v", err))
		}
	}

	// 点击发布
	utils.InfoWithPlatform(u.platform, "准备发布...")
	if err := u.publish(page, browserCtx); err != nil {
		return fmt.Errorf("发布失败: %w", err)
	}

	utils.SuccessWithPlatform(u.platform, "发布成功")
	return nil
}

// handleNewFeatureGuide 处理新功能引导
func (u *Uploader) handleNewFeatureGuide(page playwright.Page) error {
	newFeatureBtn := page.Locator("button[type='button'] span:has-text('我知道了')")
	count, _ := newFeatureBtn.Count()
	if count > 0 {
		utils.InfoWithPlatform(u.platform, "检测到新功能引导，点击'我知道了'...")
		if err := newFeatureBtn.Click(); err == nil {
			time.Sleep(1 * time.Second)
		}
	}
	return nil
}

// uploadVideo 上传视频
func (u *Uploader) uploadVideo(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext, videoPath string) error {
	utils.InfoWithPlatform(u.platform, "正在上传视频...")

	uploadButton := page.Locator("button[class^='_upload-btn']")
	if err := uploadButton.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(10000),
	}); err != nil {
		return fmt.Errorf("上传按钮不可见: %w", err)
	}

	fileChooser, err := page.ExpectFileChooser(func() error {
		return uploadButton.Click()
	})
	if err != nil {
		return fmt.Errorf("等待文件选择器失败: %w", err)
	}

	if err := fileChooser.SetFiles(videoPath); err != nil {
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
	maxRetries := 60
	retryInterval := 2 * time.Second

	for retryCount := 0; retryCount < maxRetries; retryCount++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("上传已取消")
		default:
		}

		if browserCtx.IsPageClosed() {
			return fmt.Errorf("浏览器已关闭")
		}

		// 检测方式1："上传中"文本消失
		uploadingCount, _ := page.Locator("text=上传中").Count()
		if uploadingCount == 0 {
			// 检查是否有成功标志
			successCount, _ := page.Locator("[class*='success'] >> text=上传成功").Count()
			if successCount > 0 {
				utils.InfoWithPlatform(u.platform, "视频上传成功")
				return nil
			}
			// 检查视频预览是否出现
			videoPreview := page.Locator("video, .video-preview, [class*='videoPreview']").First()
			if count, _ := videoPreview.Count(); count > 0 {
				if visible, _ := videoPreview.IsVisible(); visible {
					utils.InfoWithPlatform(u.platform, "视频上传完成（检测到预览）")
					return nil
				}
			}
			utils.InfoWithPlatform(u.platform, "视频上传完成")
			return nil
		}

		// 检测上传失败
		errorText := page.Locator("text=/上传失败|上传出错|Upload failed/").First()
		if count, _ := errorText.Count(); count > 0 {
			return fmt.Errorf("视频上传失败")
		}

		if retryCount%5 == 0 {
			utils.InfoWithPlatform(u.platform, fmt.Sprintf("正在上传视频中... (%d/%d)", retryCount, maxRetries))
		}

		time.Sleep(retryInterval)
	}

	return fmt.Errorf("上传超时，已等待%d次检测", maxRetries)
}

// fillDescription 填写描述和标题
func (u *Uploader) fillDescription(page playwright.Page, title, description string) error {
	utils.InfoWithPlatform(u.platform, "填写描述...")

	// 定位描述输入区域
	descArea := page.GetByText("描述").Locator("xpath=following-sibling::div")
	if err := descArea.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		// 兜底：尝试其他选择器
		descArea = page.Locator("textarea[placeholder*='描述'], div[contenteditable='true']").First()
		if err := descArea.WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(3000),
		}); err != nil {
			return fmt.Errorf("未找到描述输入区域: %w", err)
		}
	}

	// 点击并清空
	if err := descArea.Click(); err != nil {
		return fmt.Errorf("点击描述区域失败: %w", err)
	}
	time.Sleep(300 * time.Millisecond)

	page.Keyboard().Press("Backspace")
	page.Keyboard().Press("Control+KeyA")
	page.Keyboard().Press("Delete")
	time.Sleep(300 * time.Millisecond)

	// 填写标题（快手描述区域通常包含标题）
	content := ""
	if title != "" {
		content = title
	}
	if description != "" {
		if content != "" {
			content += "\n"
		}
		content += description
	}

	if content != "" {
		page.Keyboard().Type(content)
		utils.InfoWithPlatform(u.platform, "描述已填写")
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}

// addTags 添加话题标签（最多3个）
func (u *Uploader) addTags(page playwright.Page, tags []string) error {
	// 快手最多支持3个标签
	maxTags := 3
	if len(tags) < maxTags {
		maxTags = len(tags)
	}
	tagsToAdd := tags[:maxTags]

	utils.InfoWithPlatform(u.platform, fmt.Sprintf("添加%d个标签（最多3个）...", len(tagsToAdd)))

	for _, tag := range tagsToAdd {
		// 清理标签
		cleanTag := strings.TrimSpace(tag)
		cleanTag = strings.ReplaceAll(cleanTag, "#", "")

		if cleanTag == "" {
			continue
		}

		// 在描述区域输入标签
		page.Keyboard().Type(fmt.Sprintf("#%s ", cleanTag))
		time.Sleep(2 * time.Second) // 快手标签需要等待联想
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

	// 查找封面设置按钮
	coverBtn := page.GetByText("设置封面").First()
	if err := coverBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		// 尝试其他选择器
		coverBtn = page.Locator("[class*='cover'], .cover-setting").First()
		if err := coverBtn.WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(3000),
		}); err != nil {
			return fmt.Errorf("未找到封面设置按钮: %w", err)
		}
	}

	if err := coverBtn.Click(); err != nil {
		return fmt.Errorf("点击封面设置按钮失败: %w", err)
	}
	time.Sleep(2 * time.Second)

	// 查找文件输入框
	coverInput := page.Locator("input[type='file'][accept*='image']").First()
	if err := coverInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("未找到封面文件输入框: %w", err)
	}

	if err := coverInput.SetInputFiles(coverPath); err != nil {
		return fmt.Errorf("上传封面失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "封面上传中...")
	time.Sleep(3 * time.Second)

	// 点击确认按钮
	confirmBtn := page.Locator("button:has-text('确认'), button:has-text('完成'), button:has-text('确定')").First()
	if count, _ := confirmBtn.Count(); count > 0 {
		if err := confirmBtn.Click(); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("点击确认按钮失败: %v", err))
		}
		time.Sleep(1 * time.Second)
	}

	utils.InfoWithPlatform(u.platform, "封面设置完成")
	return nil
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
	scheduleLabel := page.Locator("label:has-text('发布时间')")
	if err := scheduleLabel.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(3000),
	}); err != nil {
		return fmt.Errorf("未找到发布时间选项: %w", err)
	}

	// 选择定时发布单选框（通常是第二个）
	scheduleRadio := scheduleLabel.Locator("xpath=following-sibling::div").Locator(".ant-radio-input").Nth(1)
	if err := scheduleRadio.Click(); err != nil {
		// 兜底：直接点击"定时发布"文本
		scheduleText := page.GetByText("定时发布")
		if err := scheduleText.Click(); err != nil {
			return fmt.Errorf("点击定时发布失败: %w", err)
		}
	}
	time.Sleep(1 * time.Second)

	// 选择时间
	scheduleInput := page.Locator("div.ant-picker-input input[placeholder*='选择日期时间']")
	if err := scheduleInput.Click(); err != nil {
		scheduleInput = page.Locator("input[placeholder*='时间']")
		if err := scheduleInput.Click(); err != nil {
			return fmt.Errorf("点击时间输入框失败: %w", err)
		}
	}
	time.Sleep(1 * time.Second)

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
func (u *Uploader) publish(page playwright.Page, browserCtx *browser.PooledContext) error {
	maxAttempts := 30

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if browserCtx.IsPageClosed() {
			return fmt.Errorf("浏览器已关闭")
		}

		// 定位发布按钮
		publishButton := page.GetByText("发布", playwright.PageGetByTextOptions{Exact: playwright.Bool(true)})
		count, _ := publishButton.Count()
		if count > 0 {
			if err := publishButton.Click(); err != nil {
				utils.WarnWithPlatform(u.platform, fmt.Sprintf("点击发布按钮失败: %v", err))
			}
		}

		time.Sleep(1 * time.Second)

		// 处理确认弹窗
		confirmButton := page.GetByText("确认发布")
		confirmCount, _ := confirmButton.Count()
		if confirmCount > 0 {
			if err := confirmButton.Click(); err != nil {
				utils.WarnWithPlatform(u.platform, fmt.Sprintf("点击确认发布失败: %v", err))
			}
		}

		// 检测发布结果
		currentURL := page.URL()
		if currentURL == "https://cp.kuaishou.com/article/manage/video?status=2&from=publish" {
			utils.InfoWithPlatform(u.platform, "发布成功，页面已跳转")
			return nil
		}

		// 检测成功提示
		successCount, _ := page.Locator("text=发布成功").Count()
		if successCount > 0 {
			if visible, _ := page.Locator("text=发布成功").IsVisible(); visible {
				utils.InfoWithPlatform(u.platform, "发布成功")
				return nil
			}
		}

		// 检测错误
		errorText := page.Locator("text=/发布失败|提交失败|错误/").First()
		if count, _ := errorText.Count(); count > 0 {
			if visible, _ := errorText.IsVisible(); visible {
				text, _ := errorText.TextContent()
				return fmt.Errorf("发布出错: %s", text)
			}
		}

		utils.InfoWithPlatform(u.platform, fmt.Sprintf("正在发布中... (尝试 %d/%d)", attempt+1, maxAttempts))
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("发布超时，已尝试%d次", maxAttempts)
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

	utils.InfoWithPlatform(u.platform, "正在打开创作者中心...")
	if _, err := page.Goto("https://cp.kuaishou.com/article/publish/video", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		return fmt.Errorf("打开登录页面失败: %w", err)
	}

	time.Sleep(3 * time.Second)

	utils.InfoWithPlatform(u.platform, "请在浏览器窗口中完成登录，登录成功后会自动保存")

	// 使用Cookie检测机制等待登录成功
	cookieConfig, ok := browser.GetCookieConfig("kuaishou")
	if !ok {
		return fmt.Errorf("获取快手Cookie配置失败")
	}

	if err := browserCtx.WaitForLoginCookies(cookieConfig); err != nil {
		return fmt.Errorf("等待登录Cookie失败: %w", err)
	}

	utils.SuccessWithPlatform(u.platform, "登录成功，检测到kuaishou.server.web_ph Cookie")
	if err := browserCtx.SaveCookiesTo(u.cookiePath); err != nil {
		return fmt.Errorf("保存Cookie失败: %w", err)
	}
	utils.InfoWithPlatform(u.platform, "Cookie已保存")
	return nil
}
