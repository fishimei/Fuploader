package tiktok

import (
	"context"
	"fmt"
	"os"
	"strconv"
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
		utils.InfoWithPlatform("tiktok", fmt.Sprintf("[调试] "+format, args...))
	}
}

// browserPool 全局浏览器池实例
var browserPool *browser.Pool

func init() {
	browserPool = browser.NewPool(2, 5)
}

// Uploader TikTok上传器
type Uploader struct {
	cookiePath string
	platform   string
}

// NewUploader 创建上传器
func NewUploader(cookiePath string) *Uploader {
	u := &Uploader{
		cookiePath: cookiePath,
		platform:   "tiktok",
	}
	debugLog("创建上传器 - 地址: %p, cookiePath: '%s'", u, cookiePath)
	if cookiePath == "" {
		utils.Warn("[TikTok] NewUploader 收到空的cookiePath!")
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

	browserCtx, err := browserPool.GetContext(ctx, u.cookiePath, u.getContextOptions())
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

	if _, err := page.Goto("https://www.tiktok.com/tiktokstudio/upload?lang=en", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("打开页面失败: %v", err))
		return false, nil
	}

	time.Sleep(3 * time.Second)

	// 使用Cookie检测机制验证登录状态
	cookieConfig, ok := browser.GetCookieConfig("tiktok")
	if !ok {
		return false, fmt.Errorf("获取TikTok Cookie配置失败")
	}

	isValid, err := browserCtx.ValidateLoginCookies(cookieConfig)
	if err != nil {
		return false, fmt.Errorf("验证Cookie失败: %w", err)
	}

	if isValid {
		utils.InfoWithPlatform(u.platform, "检测到sessionid Cookie，验证通过")
	} else {
		utils.InfoWithPlatform(u.platform, "未检测到sessionid Cookie，验证失败")
	}

	return isValid, nil
}

// Upload 上传视频
func (u *Uploader) Upload(ctx context.Context, task *types.VideoTask) error {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("开始上传: %s", task.VideoPath))

	if _, err := os.Stat(task.VideoPath); err != nil {
		return fmt.Errorf("视频文件不存在: %w", err)
	}

	browserCtx, err := browserPool.GetContext(ctx, u.cookiePath, u.getContextOptions())
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
	if _, err := page.Goto("https://www.tiktok.com/tiktokstudio/upload", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		return fmt.Errorf("打开上传页面失败: %w", err)
	}
	time.Sleep(3 * time.Second)

	// 检测iframe结构
	var locatorBase playwright.Locator
	iframeCount, _ := page.Locator("iframe[data-tt='Upload_index_iframe']").Count()
	if iframeCount > 0 {
		frame := page.FrameLocator("iframe[data-tt='Upload_index_iframe']")
		locatorBase = frame.Locator("div.upload-container")
		utils.InfoWithPlatform(u.platform, "检测到iframe结构")
	} else {
		locatorBase = page.Locator("div.upload-container")
		utils.InfoWithPlatform(u.platform, "使用普通容器结构")
	}

	// 上传视频
	if err := u.uploadVideo(ctx, page, browserCtx, locatorBase, task.VideoPath); err != nil {
		return fmt.Errorf("上传视频失败: %w", err)
	}

	time.Sleep(2 * time.Second)

	// 填写标题和描述
	if err := u.fillTitleAndDescription(locatorBase, task.Title, task.Description); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("填写标题和描述失败: %v", err))
	}

	// 添加话题标签
	if len(task.Tags) > 0 {
		if err := u.addTags(locatorBase, task.Tags); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("添加标签失败: %v", err))
		}
	}

	// 设置封面
	if task.Thumbnail != "" {
		if err := u.setCover(page, locatorBase, task.Thumbnail); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("设置封面失败: %v", err))
		}
	}

	// 设置定时发布
	if task.ScheduleTime != nil && *task.ScheduleTime != "" {
		if err := u.setScheduleTime(locatorBase, *task.ScheduleTime); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("设置定时发布失败: %v", err))
		}
	}

	// 点击发布
	utils.InfoWithPlatform(u.platform, "准备发布...")
	if err := u.publish(page, locatorBase, browserCtx); err != nil {
		return fmt.Errorf("发布失败: %w", err)
	}

	utils.SuccessWithPlatform(u.platform, "发布成功")
	return nil
}

// uploadVideo 上传视频
func (u *Uploader) uploadVideo(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext, locatorBase playwright.Locator, videoPath string) error {
	utils.InfoWithPlatform(u.platform, "正在上传视频...")

	uploadButton := locatorBase.Locator("button:has-text('Select video'):visible")
	if err := uploadButton.WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateVisible,
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
	if err := u.waitForUploadComplete(ctx, page, browserCtx, locatorBase); err != nil {
		return err
	}

	return nil
}

// waitForUploadComplete 等待视频上传完成
func (u *Uploader) waitForUploadComplete(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext, locatorBase playwright.Locator) error {
	uploadTimeout := 5 * time.Minute
	uploadCheckInterval := 2 * time.Second
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

		// 检测方式1：发布按钮可用（disabled属性为空或false）
		postButton := locatorBase.Locator("div.btn-post > button")
		disabledAttr, _ := postButton.GetAttribute("disabled")
		if disabledAttr == "" || disabledAttr == "false" {
			utils.InfoWithPlatform(u.platform, "视频上传完成（发布按钮可用）")
			return nil
		}

		// 检测方式2：检查视频预览
		videoPreview := locatorBase.Locator("video, .video-preview").First()
		if count, _ := videoPreview.Count(); count > 0 {
			if visible, _ := videoPreview.IsVisible(); visible {
				utils.InfoWithPlatform(u.platform, "视频上传完成（检测到预览）")
				return nil
			}
		}

		// 检测上传错误并重试
		selectFileBtn := locatorBase.Locator("button[aria-label='Select file']")
		if count, _ := selectFileBtn.Count(); count > 0 {
			utils.WarnWithPlatform(u.platform, "检测到上传错误，正在重试...")
			_, err := page.ExpectFileChooser(func() error {
				return selectFileBtn.Click()
			})
			if err != nil {
				return fmt.Errorf("重试文件选择失败: %w", err)
			}
			// 这里需要重新获取视频路径，简化处理
		}

		time.Sleep(uploadCheckInterval)
	}

	return fmt.Errorf("上传超时")
}

// fillTitleAndDescription 填写标题和描述
func (u *Uploader) fillTitleAndDescription(locatorBase playwright.Locator, title, description string) error {
	utils.InfoWithPlatform(u.platform, "填写标题和描述...")

	editorLocator := locatorBase.Locator("div.public-DraftEditor-content")
	if err := editorLocator.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("未找到编辑器: %w", err)
	}

	if err := editorLocator.Click(); err != nil {
		return fmt.Errorf("点击编辑器失败: %w", err)
	}
	time.Sleep(500 * time.Millisecond)

	// 清空编辑器
	page, err := editorLocator.Page()
	if err != nil {
		return fmt.Errorf("获取页面失败: %w", err)
	}
	page.Keyboard().Press("End")
	page.Keyboard().Press("Control+KeyA")
	page.Keyboard().Press("Delete")
	page.Keyboard().Press("End")
	time.Sleep(500 * time.Millisecond)

	// 输入标题
	content := title
	if description != "" {
		content += "\n\n" + description
	}

	page.Keyboard().Type(content)
	time.Sleep(500 * time.Millisecond)
	page.Keyboard().Press("End")
	page.Keyboard().Press("Enter")

	utils.InfoWithPlatform(u.platform, "标题和描述已填写")
	return nil
}

// addTags 添加话题标签
func (u *Uploader) addTags(locatorBase playwright.Locator, tags []string) error {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("添加%d个标签...", len(tags)))

	page, err := locatorBase.Page()
	if err != nil {
		return fmt.Errorf("获取页面失败: %w", err)
	}

	for _, tag := range tags {
		cleanTag := strings.TrimSpace(tag)
		cleanTag = strings.ReplaceAll(cleanTag, "#", "")
		if cleanTag == "" {
			continue
		}

		page.Keyboard().Press("End")
		time.Sleep(500 * time.Millisecond)
		page.Keyboard().Type("#" + cleanTag + " ")
		page.Keyboard().Press("Space")
		time.Sleep(500 * time.Millisecond)
		page.Keyboard().Press("Backspace")
		page.Keyboard().Press("End")
	}

	utils.InfoWithPlatform(u.platform, "标签添加完成")
	return nil
}

// setCover 设置封面
func (u *Uploader) setCover(page playwright.Page, locatorBase playwright.Locator, coverPath string) error {
	if _, err := os.Stat(coverPath); err != nil {
		return fmt.Errorf("封面文件不存在: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "设置封面...")

	// 点击封面区域
	coverContainer := locatorBase.Locator(".cover-container").First()
	if err := coverContainer.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("未找到封面区域: %w", err)
	}

	if err := coverContainer.Click(); err != nil {
		return fmt.Errorf("点击封面区域失败: %w", err)
	}
	time.Sleep(2 * time.Second)

	// 点击"Upload cover"
	uploadCoverBtn := locatorBase.GetByText("Upload cover").First()
	if count, _ := uploadCoverBtn.Count(); count > 0 {
		uploadCoverBtn.Click()
		time.Sleep(1 * time.Second)
	}

	// 等待文件选择器并上传
	fileChooser, err := page.ExpectFileChooser(func() error {
		uploadBtn := locatorBase.Locator("button:has-text('Upload'):visible").First()
		return uploadBtn.Click()
	})
	if err != nil {
		return fmt.Errorf("等待文件选择器失败: %w", err)
	}

	if err := fileChooser.SetFiles(coverPath); err != nil {
		return fmt.Errorf("上传封面失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "封面上传中...")
	time.Sleep(3 * time.Second)

	// 点击确认
	confirmBtn := locatorBase.GetByText("Confirm").First()
	if count, _ := confirmBtn.Count(); count > 0 {
		confirmBtn.Click()
		time.Sleep(1 * time.Second)
	}

	utils.InfoWithPlatform(u.platform, "封面设置完成")
	return nil
}

// setScheduleTime 设置定时发布
func (u *Uploader) setScheduleTime(locatorBase playwright.Locator, scheduleTime string) error {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("设置定时发布时间: %s", scheduleTime))

	// 解析时间
	publishDate, err := time.Parse("2006-01-02 15:04", scheduleTime)
	if err != nil {
		return fmt.Errorf("解析时间失败: %w", err)
	}

	// 点击Schedule按钮
	scheduleBtn := locatorBase.GetByLabel("Schedule")
	if err := scheduleBtn.WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateVisible,
	}); err != nil {
		return fmt.Errorf("未找到Schedule按钮: %w", err)
	}

	scheduleBtn.Click()
	time.Sleep(1 * time.Second)

	// 选择日期
	scheduledPicker := locatorBase.Locator("div.scheduled-picker")
	calendarBtn := scheduledPicker.Locator("div.TUXInputBox").Nth(1)
	calendarBtn.Click()
	time.Sleep(500 * time.Millisecond)

	// 获取当前月份
	monthTitle := locatorBase.Locator("div.calendar-wrapper span.month-title")
	monthText, _ := monthTitle.TextContent()
	currentMonth := parseMonth(monthText)
	targetMonth := int(publishDate.Month())

	// 切换月份
	if currentMonth != targetMonth {
		arrowIndex := 0
		if currentMonth < targetMonth {
			arrows := locatorBase.Locator("div.calendar-wrapper span.arrow")
			count, _ := arrows.Count()
			arrowIndex = int(count) - 1
		}
		arrow := locatorBase.Locator("div.calendar-wrapper span.arrow").Nth(arrowIndex)
		arrow.Click()
		time.Sleep(500 * time.Millisecond)
	}

	// 选择日期
	validDays := locatorBase.Locator("div.calendar-wrapper span.day.valid")
	count, _ := validDays.Count()
	targetDay := strconv.Itoa(publishDate.Day())
	for i := 0; i < count; i++ {
		dayText, _ := validDays.Nth(i).TextContent()
		if strings.TrimSpace(dayText) == targetDay {
			validDays.Nth(i).Click()
			break
		}
	}

	// 选择时间
	timeBtn := scheduledPicker.Locator("div.TUXInputBox").Nth(0)
	timeBtn.Click()
	time.Sleep(500 * time.Millisecond)

	hourStr := publishDate.Format("15")
	hourSelector := fmt.Sprintf("span.tiktok-timepicker-left:has-text('%s')", hourStr)
	hourElement := locatorBase.Locator(hourSelector)
	hourElement.Click()
	time.Sleep(500 * time.Millisecond)

	timeBtn.Click()
	time.Sleep(500 * time.Millisecond)

	// 分钟取5的倍数
	correctMinute := int(publishDate.Minute()/5) * 5
	minuteStr := fmt.Sprintf("%02d", correctMinute)
	minuteSelector := fmt.Sprintf("span.tiktok-timepicker-right:has-text('%s')", minuteStr)
	minuteElement := locatorBase.Locator(minuteSelector)
	minuteElement.Click()

	// 关闭时间选择器
	uploadTitle := locatorBase.Locator("h1:has-text('Upload video')")
	uploadTitle.Click()

	utils.InfoWithPlatform(u.platform, fmt.Sprintf("定时发布时间设置完成: %s", scheduleTime))
	return nil
}

// publish 点击发布并检测结果
func (u *Uploader) publish(page playwright.Page, locatorBase playwright.Locator, browserCtx *browser.PooledContext) error {
	successFlagDiv := "#\\:r9\\:"

	for {
		if browserCtx.IsPageClosed() {
			return fmt.Errorf("浏览器已关闭")
		}

		publishBtn := locatorBase.Locator("div.btn-post")
		if count, _ := publishBtn.Count(); count > 0 {
			publishBtn.Click()
		}

		time.Sleep(3 * time.Second)

		// 检测成功标志
		successLocator := locatorBase.Locator(successFlagDiv)
		if visible, _ := successLocator.IsVisible(); visible {
			utils.InfoWithPlatform(u.platform, "发布成功")
			return nil
		}

		if count, _ := successLocator.Count(); count > 0 {
			utils.InfoWithPlatform(u.platform, "发布成功")
			return nil
		}

		// 检测URL跳转
		url := page.URL()
		if url == "https://www.tiktok.com/tiktokstudio/content" {
			utils.InfoWithPlatform(u.platform, "发布成功，页面已跳转")
			return nil
		}

		utils.InfoWithPlatform(u.platform, "等待发布完成...")
		time.Sleep(500 * time.Millisecond)
	}
}

// parseMonth 解析英文月份名称到数字
func parseMonth(monthName string) int {
	months := map[string]int{
		"January":   1,
		"February":  2,
		"March":     3,
		"April":     4,
		"May":       5,
		"June":      6,
		"July":      7,
		"August":    8,
		"September": 9,
		"October":   10,
		"November":  11,
		"December":  12,
	}

	if month, ok := months[monthName]; ok {
		return month
	}

	for name, month := range months {
		if strings.Contains(monthName, name) {
			return month
		}
	}

	return 0
}

// Login 登录
func (u *Uploader) Login() error {
	debugLog("Login开始 - cookiePath: '%s'", u.cookiePath)
	if u.cookiePath == "" {
		return fmt.Errorf("cookie路径为空")
	}

	ctx := context.Background()
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("Cookie保存路径: %s", u.cookiePath))

	browserCtx, err := browserPool.GetContext(ctx, "", u.getContextOptions())
	if err != nil {
		return fmt.Errorf("获取浏览器失败: %w", err)
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		return fmt.Errorf("获取页面失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "正在打开登录页面...")
	if _, err := page.Goto("https://www.tiktok.com/login?lang=en", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		return fmt.Errorf("打开登录页面失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "请在浏览器窗口中完成登录")

	// 使用Cookie检测机制等待登录成功
	cookieConfig, ok := browser.GetCookieConfig("tiktok")
	if !ok {
		return fmt.Errorf("获取TikTok Cookie配置失败")
	}

	if err := browserCtx.WaitForLoginCookies(cookieConfig); err != nil {
		return fmt.Errorf("等待登录Cookie失败: %w", err)
	}

	utils.SuccessWithPlatform(u.platform, "登录成功，检测到sessionid Cookie")
	if err := browserCtx.SaveCookiesTo(u.cookiePath); err != nil {
		return fmt.Errorf("保存Cookie失败: %w", err)
	}
	utils.InfoWithPlatform(u.platform, "Cookie已保存")
	return nil
}

// getContextOptions 获取TikTok特定的上下文选项
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
