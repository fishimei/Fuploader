package douyin

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
		utils.InfoWithPlatform("douyin", fmt.Sprintf("[调试] "+format, args...))
	}
}

// browserPool 全局浏览器池实例
var browserPool *browser.Pool

func init() {
	browserPool = browser.NewPool(2, 5)
}

// Uploader 抖音上传器
type Uploader struct {
	cookiePath string
	platform   string
}

// NewUploader 创建上传器
func NewUploader(cookiePath string) *Uploader {
	u := &Uploader{
		cookiePath: cookiePath,
		platform:   "douyin",
	}
	debugLog("创建上传器 - 地址: %p, cookiePath: '%s'", u, cookiePath)
	if cookiePath == "" {
		utils.Warn("[Douyin] NewUploader 收到空的cookiePath!")
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

	if _, err := page.Goto("https://creator.douyin.com/", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("打开页面失败: %v", err))
		return false, nil
	}

	time.Sleep(3 * time.Second)

	// 使用Cookie检测机制验证登录状态
	cookieConfig, ok := browser.GetCookieConfig("douyin")
	if !ok {
		return false, fmt.Errorf("获取抖音Cookie配置失败")
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
	if _, err := page.Goto("https://creator.douyin.com/creator-micro/content/upload", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		return fmt.Errorf("打开上传页面失败: %w", err)
	}
	time.Sleep(3 * time.Second)

	// 上传视频
	utils.InfoWithPlatform(u.platform, "上传视频中...")
	if err := u.uploadVideo(ctx, page, browserCtx, task.VideoPath); err != nil {
		return fmt.Errorf("上传视频失败: %w", err)
	}

	time.Sleep(2 * time.Second)

	// 填写标题（限制30字符）
	if task.Title != "" {
		utils.InfoWithPlatform(u.platform, "填写标题...")
		if err := u.fillTitle(page, task.Title); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("填写标题失败: %v", err))
		}
	}

	// 填写描述
	if task.Description != "" {
		utils.InfoWithPlatform(u.platform, "填写描述...")
		if err := u.fillDescription(page, task.Description); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("填写描述失败: %v", err))
		}
	}

	// 添加话题标签
	if len(task.Tags) > 0 {
		utils.InfoWithPlatform(u.platform, fmt.Sprintf("添加%d个标签...", len(task.Tags)))
		if err := u.addTags(page, task.Tags); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("添加标签失败: %v", err))
		}
	}

	// 设置封面（始终执行，有自定义封面时上传，无则使用默认）
	if err := u.setCover(page, task.Thumbnail); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("设置封面失败: %v", err))
	}

	// 添加商品链接
	if task.ProductLink != "" {
		utils.InfoWithPlatform(u.platform, "添加商品链接...")
		if err := u.addProductLink(page, task.ProductLink, task.ProductTitle); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("添加商品链接失败: %v", err))
		}
	}

	// 设置同步选项
	if task.SyncToutiao || task.SyncXigua {
		utils.InfoWithPlatform(u.platform, "设置同步选项...")
		if err := u.setSyncOptions(page, task.SyncToutiao, task.SyncXigua); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("设置同步选项失败: %v", err))
		}
	}

	// 设置定时发布
	if task.ScheduleTime != nil && *task.ScheduleTime != "" {
		utils.InfoWithPlatform(u.platform, fmt.Sprintf("设置定时发布: %s", *task.ScheduleTime))
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

// uploadVideo 上传视频
func (u *Uploader) uploadVideo(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext, videoPath string) error {
	utils.InfoWithPlatform(u.platform, "正在上传视频...")

	// 定位文件输入框
	inputLocator := page.Locator("div[class^='container'] input[type='file']").First()
	if err := inputLocator.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(10000),
	}); err != nil {
		// 兜底：尝试通用选择器
		inputLocator = page.Locator("input[type='file']").First()
		if err := inputLocator.WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(5000),
		}); err != nil {
			return fmt.Errorf("未找到文件输入框: %w", err)
		}
	}

	if err := inputLocator.SetInputFiles(videoPath); err != nil {
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
	uploadTimeout := 10 * time.Minute
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

		// 检测方式1：视频预览区域出现
		videoPreview := page.Locator("video, .video-preview, [class*='videoPreview'], div[class*='player']").First()
		if count, _ := videoPreview.Count(); count > 0 {
			if visible, _ := videoPreview.IsVisible(); visible {
				utils.InfoWithPlatform(u.platform, "视频上传完成（检测到预览）")
				return nil
			}
		}

		// 检测方式2：上传进度条消失
		progressBar := page.Locator("div[class*='progress'], div[class*='uploading']").First()
		if count, _ := progressBar.Count(); count == 0 {
			// 检查是否有视频信息
			videoInfo := page.Locator("div[class*='video-info'], div[class*='mediaInfo']").First()
			if count, _ := videoInfo.Count(); count > 0 {
				utils.InfoWithPlatform(u.platform, "视频上传完成")
				return nil
			}
		}

		// 检测方式3："上传成功"文本
		successText := page.Locator("text=/上传成功|上传完成/").First()
		if count, _ := successText.Count(); count > 0 {
			if visible, _ := successText.IsVisible(); visible {
				utils.InfoWithPlatform(u.platform, "视频上传完成（检测到成功文本）")
				return nil
			}
		}

		// 检测上传失败
		errorText := page.Locator("text=/上传失败|上传出错/").First()
		if count, _ := errorText.Count(); count > 0 {
			return fmt.Errorf("视频上传失败")
		}

		time.Sleep(uploadCheckInterval)
	}

	return fmt.Errorf("上传超时")
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

	// 尝试多种选择器定位标题输入框
	titleContainer := page.GetByText("作品标题").Locator("..").Locator("xpath=following-sibling::div[1]").Locator("input")
	if err := titleContainer.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(3000),
	}); err != nil {
		// 兜底：尝试其他选择器
		titleContainer = page.Locator(".notranslate").First()
		if err := titleContainer.WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(3000),
		}); err != nil {
			return fmt.Errorf("未找到标题输入框: %w", err)
		}
	}

	count, _ := titleContainer.Count()
	if count > 0 {
		if err := titleContainer.Fill(title); err != nil {
			// 兜底：使用键盘输入
			titleContainer.Click()
			page.Keyboard().Press("Control+KeyA")
			page.Keyboard().Press("Delete")
			page.Keyboard().Type(title)
		}
	} else {
		// 兜底方案
		titleInput := page.Locator(".notranslate").First()
		if count, _ := titleInput.Count(); count > 0 {
			titleInput.Click()
			page.Keyboard().Press("Control+KeyA")
			page.Keyboard().Press("Delete")
			page.Keyboard().Type(title)
			page.Keyboard().Press("Enter")
		}
	}

	utils.InfoWithPlatform(u.platform, fmt.Sprintf("标题已填写: %s", title))
	time.Sleep(500 * time.Millisecond)
	return nil
}

// fillDescription 填写描述
func (u *Uploader) fillDescription(page playwright.Page, description string) error {
	utils.InfoWithPlatform(u.platform, "填写描述...")

	// 定位描述输入框
	descContainer := page.GetByText("作品描述").Locator("..").Locator("xpath=following-sibling::div[1]").Locator("textarea, .notranslate").First()
	if err := descContainer.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(3000),
	}); err != nil {
		// 兜底：尝试其他选择器
		descContainer = page.Locator(".notranslate").Nth(1)
		if err := descContainer.WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(3000),
		}); err != nil {
			return fmt.Errorf("未找到描述输入框: %w", err)
		}
	}

	count, _ := descContainer.Count()
	if count > 0 {
		if err := descContainer.Fill(description); err != nil {
			// 兜底：使用键盘输入
			descContainer.Click()
			page.Keyboard().Press("Control+KeyA")
			page.Keyboard().Press("Delete")
			page.Keyboard().Type(description)
		}
	} else {
		return fmt.Errorf("未找到描述输入框")
	}

	utils.InfoWithPlatform(u.platform, "描述已填写")
	time.Sleep(500 * time.Millisecond)
	return nil
}

// addTags 添加话题标签
func (u *Uploader) addTags(page playwright.Page, tags []string) error {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("添加%d个标签...", len(tags)))

	// 定位标签输入区域
	tagContainer := page.Locator(".zone-container").First()
	if err := tagContainer.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(3000),
	}); err != nil {
		// 兜底：尝试其他选择器
		tagContainer = page.Locator("div[class*='tag'], div[class*='topic']").First()
		if err := tagContainer.WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(3000),
		}); err != nil {
			return fmt.Errorf("未找到标签输入区域: %w", err)
		}
	}

	count, _ := tagContainer.Count()
	if count > 0 {
		for _, tag := range tags {
			cleanTag := strings.TrimSpace(tag)
			cleanTag = strings.ReplaceAll(cleanTag, "#", "")
			if cleanTag == "" {
				continue
			}

			tagContainer.Type("#"+cleanTag, playwright.LocatorTypeOptions{Delay: playwright.Float(100)})
			tagContainer.Press("Space")
			time.Sleep(300 * time.Millisecond)
		}
	}

	utils.InfoWithPlatform(u.platform, "标签添加完成")
	return nil
}

// setCover 设置封面
func (u *Uploader) setCover(page playwright.Page, coverPath string) error {
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

	// 上传自定义封面
	if coverPath != "" {
		if _, err := os.Stat(coverPath); err == nil {
			utils.InfoWithPlatform(u.platform, "上传自定义封面...")

			// 直接定位隐藏的文件输入框（使用class选择器更可靠）
			coverInput := page.Locator(`input.semi-upload-hidden-input[type="file"]`).First()
			if count, _ := coverInput.Count(); count == 0 {
				// 兜底：通用选择器
				coverInput = page.Locator(`input[type="file"]`).First()
			}

			if count, _ := coverInput.Count(); count > 0 {
				if err := coverInput.SetInputFiles(coverPath); err != nil {
					utils.WarnWithPlatform(u.platform, fmt.Sprintf("上传封面失败: %v", err))
				} else {
					utils.InfoWithPlatform(u.platform, "封面上传中...")
					time.Sleep(3 * time.Second)
				}
			}
		}
	}

	// 点击"设置竖封面"按钮切换到竖封面界面
	verticalBtn := page.Locator(`button:has-text("设置竖封面")`).First()
	if err := verticalBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		utils.WarnWithPlatform(u.platform, "未找到设置竖封面按钮")
	} else {
		if err := verticalBtn.Click(); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("点击设置竖封面按钮失败: %v", err))
		} else {
			utils.InfoWithPlatform(u.platform, "已切换到竖封面")
		}
		time.Sleep(2 * time.Second)
	}

	// 点击完成按钮
	finishBtn := page.Locator(`button:has-text("完成")`).First()
	if err := finishBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		// 兜底：尝试其他完成按钮选择器
		finishBtn = page.Locator(`div#tooltip-container button:visible:has-text("完成")`).First()
		if count, _ := finishBtn.Count(); count == 0 {
			finishBtn = page.Locator(`div[class^="extractFooter"] button:has-text("完成")`).First()
		}
	}

	if count, _ := finishBtn.Count(); count > 0 {
		if err := finishBtn.Click(); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("点击完成按钮失败: %v", err))
		} else {
			utils.InfoWithPlatform(u.platform, "已点击完成按钮")
		}
	}
	time.Sleep(2 * time.Second)

	utils.InfoWithPlatform(u.platform, "封面设置完成")
	return nil
}

// addProductLink 添加商品链接
func (u *Uploader) addProductLink(page playwright.Page, productLink, productTitle string) error {
	utils.InfoWithPlatform(u.platform, "添加商品链接...")

	// 点击"添加标签"
	addTagBtn := page.GetByText("添加标签").First()
	if err := addTagBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(3000),
	}); err != nil {
		return fmt.Errorf("未找到添加标签按钮: %w", err)
	}

	if err := addTagBtn.Click(); err != nil {
		return fmt.Errorf("点击添加标签失败: %w", err)
	}
	time.Sleep(1 * time.Second)

	// 点击"购物车"
	cartBtn := page.GetByText("购物车").First()
	if count, _ := cartBtn.Count(); count > 0 {
		cartBtn.Click()
		time.Sleep(1 * time.Second)
	}

	// 填写商品链接
	linkInput := page.Locator("input[placeholder='添加商品链接']").First()
	if err := linkInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(3000),
	}); err != nil {
		return fmt.Errorf("未找到商品链接输入框: %w", err)
	}

	if err := linkInput.Fill(productLink); err != nil {
		return fmt.Errorf("填写商品链接失败: %w", err)
	}

	// 填写商品短标题（限制10字符）
	if productTitle != "" {
		shortTitle := productTitle
		if len(shortTitle) > 10 {
			runes := []rune(shortTitle)
			if len(runes) > 10 {
				shortTitle = string(runes[:10])
			}
		}

		titleInput := page.Locator("input[placeholder*='短标题']").First()
		if count, _ := titleInput.Count(); count > 0 {
			titleInput.Fill(shortTitle)
		}
	}

	utils.InfoWithPlatform(u.platform, "商品链接添加完成")
	time.Sleep(1 * time.Second)
	return nil
}

// setSyncOptions 设置同步选项
func (u *Uploader) setSyncOptions(page playwright.Page, syncToutiao, syncXigua bool) error {
	utils.InfoWithPlatform(u.platform, "设置同步选项...")

	// 同步到今日头条
	if syncToutiao {
		toutiaoCheckbox := page.Locator("text=同步到今日头条").Locator("xpath=../input[type='checkbox']").First()
		if count, _ := toutiaoCheckbox.Count(); count > 0 {
			isChecked, _ := toutiaoCheckbox.IsChecked()
			if !isChecked {
				toutiaoCheckbox.Check()
				utils.InfoWithPlatform(u.platform, "已勾选同步到今日头条")
			}
		}
	}

	// 同步到西瓜视频
	if syncXigua {
		xiguaCheckbox := page.Locator("text=同步到西瓜视频").Locator("xpath=../input[type='checkbox']").First()
		if count, _ := xiguaCheckbox.Count(); count > 0 {
			isChecked, _ := xiguaCheckbox.IsChecked()
			if !isChecked {
				xiguaCheckbox.Check()
				utils.InfoWithPlatform(u.platform, "已勾选同步到西瓜视频")
			}
		}
	}

	time.Sleep(500 * time.Millisecond)
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
	scheduleLabel := page.GetByText("定时发布").First()
	if err := scheduleLabel.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(3000),
	}); err != nil {
		return fmt.Errorf("未找到定时发布选项: %w", err)
	}

	if err := scheduleLabel.Click(); err != nil {
		return fmt.Errorf("点击定时发布失败: %w", err)
	}
	time.Sleep(1 * time.Second)

	// 选择日期
	datePicker := page.Locator("input[placeholder*='日期'], div[class*='date-picker']").First()
	if err := datePicker.Click(); err != nil {
		return fmt.Errorf("点击日期选择器失败: %w", err)
	}
	time.Sleep(500 * time.Millisecond)

	// 选择目标日期
	dateStr := targetTime.Format("2006-01-02")
	dateCell := page.Locator(fmt.Sprintf("td[title='%s'], td:has-text('%d')", dateStr, targetTime.Day())).First()
	if count, _ := dateCell.Count(); count > 0 {
		dateCell.Click()
	}
	time.Sleep(500 * time.Millisecond)

	// 选择时间
	timePicker := page.Locator("input[placeholder*='时间']").First()
	if err := timePicker.Click(); err != nil {
		return fmt.Errorf("点击时间选择器失败: %w", err)
	}
	time.Sleep(500 * time.Millisecond)

	// 输入时间
	timeStr := targetTime.Format("15:04")
	page.Keyboard().Press("Control+KeyA")
	page.Keyboard().Type(timeStr)
	page.Keyboard().Press("Enter")

	utils.InfoWithPlatform(u.platform, fmt.Sprintf("定时发布时间设置完成: %s", scheduleTime))
	time.Sleep(1 * time.Second)
	return nil
}

// publish 点击发布并检测结果
func (u *Uploader) publish(page playwright.Page, browserCtx *browser.PooledContext) error {
	maxRetries := 20

	for retryCount := 0; retryCount < maxRetries; retryCount++ {
		if browserCtx.IsPageClosed() {
			return fmt.Errorf("浏览器已关闭")
		}

		// 定位发布按钮
		publishBtn := page.GetByRole("button", playwright.PageGetByRoleOptions{
			Name:  "发布",
			Exact: playwright.Bool(true),
		})
		if count, _ := publishBtn.Count(); count > 0 {
			if err := publishBtn.Click(); err != nil {
				utils.WarnWithPlatform(u.platform, fmt.Sprintf("点击发布按钮失败: %v", err))
			}
		}

		time.Sleep(5 * time.Second)

		// 检测发布结果
		url := page.URL()
		if strings.Contains(url, "creator.douyin.com/creator-micro/content/manage") {
			utils.InfoWithPlatform(u.platform, "发布成功，页面已跳转")
			return nil
		}

		// 处理封面未设置提示
		coverPrompt := page.GetByText("请设置封面后再发布").First()
		if visible, _ := coverPrompt.IsVisible(); visible {
			utils.InfoWithPlatform(u.platform, "检测到封面未设置，尝试选择推荐封面...")
			recommendCover := page.Locator("[class^='recommendCover-']").First()
			if count, _ := recommendCover.Count(); count > 0 {
				recommendCover.Click()
				time.Sleep(1 * time.Second)
				confirmBtn := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "确定"})
				if count, _ := confirmBtn.Count(); count > 0 {
					confirmBtn.Click()
					time.Sleep(1 * time.Second)
				}
			}
		}

		// 检测成功提示
		successText := page.Locator("text=/发布成功|提交成功/").First()
		if count, _ := successText.Count(); count > 0 {
			if visible, _ := successText.IsVisible(); visible {
				utils.InfoWithPlatform(u.platform, "发布成功")
				return nil
			}
		}

		utils.InfoWithPlatform(u.platform, fmt.Sprintf("正在发布中... (尝试 %d/%d)", retryCount+1, maxRetries))
	}

	return fmt.Errorf("发布超时，已重试%d次", maxRetries)
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
	if _, err := page.Goto("https://creator.douyin.com/", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		return fmt.Errorf("打开登录页面失败: %w", err)
	}

	time.Sleep(3 * time.Second)

	utils.InfoWithPlatform(u.platform, "请在浏览器窗口中完成登录...")

	// 使用Cookie检测机制等待登录成功
	cookieConfig, ok := browser.GetCookieConfig("douyin")
	if !ok {
		return fmt.Errorf("获取抖音Cookie配置失败")
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
