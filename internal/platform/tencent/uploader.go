package tencent

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
		utils.InfoWithPlatform("tencent", fmt.Sprintf("[调试] "+format, args...))
	}
}

// browserPool 全局浏览器池实例
var browserPool *browser.Pool

func init() {
	browserPool = browser.NewPool(2, 5)
}

// Uploader 视频号上传器
type Uploader struct {
	cookiePath string
	platform   string
}

// NewUploader 创建上传器
func NewUploader(cookiePath string) *Uploader {
	u := &Uploader{
		cookiePath: cookiePath,
		platform:   "tencent",
	}
	debugLog("创建上传器 - 地址: %p, cookiePath: '%s'", u, cookiePath)
	if cookiePath == "" {
		utils.Warn("[Tencent] NewUploader 收到空的cookiePath!")
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

	if _, err := page.Goto("https://channels.weixin.qq.com/platform/post/create", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("打开页面失败: %v", err))
		return false, nil
	}

	time.Sleep(3 * time.Second)

	// 使用Cookie检测机制验证登录状态
	cookieConfig, ok := browser.GetCookieConfig("tencent")
	if !ok {
		return false, fmt.Errorf("获取视频号Cookie配置失败")
	}

	isValid, err := browserCtx.ValidateLoginCookies(cookieConfig)
	if err != nil {
		return false, fmt.Errorf("验证Cookie失败: %w", err)
	}

	if isValid {
		utils.InfoWithPlatform(u.platform, "检测到wxsid和wxuin Cookie，验证通过")
	} else {
		utils.InfoWithPlatform(u.platform, "未检测到wxsid和wxuin Cookie，验证失败")
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

	// 导航到发布页面
	utils.InfoWithPlatform(u.platform, "正在打开发布页面...")
	if _, err := page.Goto("https://channels.weixin.qq.com/platform/post/create", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(30000),
	}); err != nil {
		return fmt.Errorf("打开发布页面失败: %w", err)
	}

	// 等待页面关键元素加载（文件上传框）
	utils.InfoWithPlatform(u.platform, "等待页面渲染...")
	fileInput := page.Locator("input[type='file']")
	if err := fileInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(30000),
	}); err != nil {
		return fmt.Errorf("等待文件上传框超时: %w", err)
	}
	utils.InfoWithPlatform(u.platform, "页面渲染完成")

	// 上传视频
	if err := u.uploadVideo(ctx, page, browserCtx, task.VideoPath); err != nil {
		return fmt.Errorf("上传视频失败: %w", err)
	}

	time.Sleep(2 * time.Second)

	// 填写标题和描述
	if err := u.fillTitleAndDescription(page, task.Title, task.Description); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("填写标题和描述失败: %v", err))
	}

	// 添加话题标签
	if len(task.Tags) > 0 {
		if err := u.addTags(page, task.Tags); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("添加标签失败: %v", err))
		}
	}

	// 声明原创
	if task.IsOriginal {
		if err := u.setOriginal(page, task.OriginalType); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("设置原创声明失败: %v", err))
		}
	}

	// 添加到合集
	if task.Collection != "" {
		if err := u.addToCollection(page, task.Collection); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("添加到合集失败: %v", err))
		}
	}

	// 设置封面
	if task.Thumbnail != "" {
		if err := u.setCover(page, task.Thumbnail); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("设置封面失败: %v", err))
		}
	}

	// 设置短标题（使用标题自动格式化）
	if err := u.setShortTitle(page, task.Title); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("设置短标题失败: %v", err))
	}

	// 设置定时发布
	if task.ScheduleTime != nil && *task.ScheduleTime != "" {
		if err := u.setScheduleTime(page, *task.ScheduleTime); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("设置定时发布失败: %v", err))
		}
	}

	// 发布或保存草稿
	if task.IsDraft {
		if err := u.saveDraft(page, browserCtx); err != nil {
			return fmt.Errorf("保存草稿失败: %w", err)
		}
	} else {
		if err := u.publish(page, browserCtx); err != nil {
			return fmt.Errorf("发布失败: %w", err)
		}
	}

	return nil
}

// uploadVideo 上传视频
func (u *Uploader) uploadVideo(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext, videoPath string) error {
	utils.InfoWithPlatform(u.platform, "正在上传视频...")

	// 定位文件输入框
	fileInput := page.Locator("input[type='file']")
	if err := fileInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(10000),
	}); err != nil {
		return fmt.Errorf("未找到文件输入框: %w", err)
	}

	if err := fileInput.SetInputFiles(videoPath); err != nil {
		return fmt.Errorf("上传视频失败: %w", err)
	}

	// 等待上传完成
	utils.InfoWithPlatform(u.platform, "等待视频上传完成...")
	if err := u.waitForUploadComplete(ctx, page, browserCtx, videoPath); err != nil {
		return err
	}

	return nil
}

// waitForUploadComplete 等待视频上传完成
func (u *Uploader) waitForUploadComplete(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext, videoPath string) error {
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

		// 基于DOM截图优化：检查"发表"按钮的class（使用weui-desktop-btn_primary和weui-desktop-btn_disabled）
		publishBtn := page.Locator("button.weui-desktop-btn_primary:has-text('发表')").First()
		if count, _ := publishBtn.Count(); count > 0 {
			classAttr, _ := publishBtn.GetAttribute("class")
			if classAttr != "" && !strings.Contains(classAttr, "weui-desktop-btn_disabled") {
				utils.InfoWithPlatform(u.platform, "视频上传完成")
				return nil
			}
		}

		// 检测上传失败（Python版逻辑）
		errorMsg := page.Locator("div.status-msg.error").First()
		deleteBtn := page.Locator("div.media-status-content div.tag-inner:has-text('删除')").First()
		if errorCount, _ := errorMsg.Count(); errorCount > 0 {
			if deleteCount, _ := deleteBtn.Count(); deleteCount > 0 {
				utils.WarnWithPlatform(u.platform, "检测到上传出错，准备重试...")
				if err := u.handleUploadError(page, videoPath); err != nil {
					return fmt.Errorf("重试上传失败: %w", err)
				}
				// 重置计时器继续等待
				uploadStartTime = time.Now()
			}
		}

		utils.InfoWithPlatform(u.platform, "正在上传视频中...")
		time.Sleep(uploadCheckInterval)
	}

	return fmt.Errorf("上传超时")
}

// handleUploadError 处理上传错误并重试
func (u *Uploader) handleUploadError(page playwright.Page, videoPath string) error {
	utils.InfoWithPlatform(u.platform, "视频出错了，重新上传中...")

	// 点击删除按钮
	deleteBtn := page.Locator("div.media-status-content div.tag-inner:has-text('删除')").First()
	if err := deleteBtn.Click(); err != nil {
		return fmt.Errorf("点击删除按钮失败: %w", err)
	}

	// 确认删除
	confirmBtn := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "删除", Exact: playwright.Bool(true)}).First()
	if err := confirmBtn.Click(); err != nil {
		return fmt.Errorf("点击确认删除失败: %w", err)
	}

	// 重新上传
	fileInput := page.Locator("input[type='file']").First()
	if err := fileInput.SetInputFiles(videoPath); err != nil {
		return fmt.Errorf("重新上传视频失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "已重新上传视频")
	return nil
}

// fillTitleAndDescription 填写标题和描述
func (u *Uploader) fillTitleAndDescription(page playwright.Page, title, description string) error {
	utils.InfoWithPlatform(u.platform, "填写标题和描述...")

	// 基于DOM截图优化：使用[contenteditable][data-placeholder="添加描述"]定位编辑器
	editor := page.Locator("[contenteditable][data-placeholder='添加描述']")
	if err := editor.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("未找到编辑器: %w", err)
	}

	if err := editor.Click(); err != nil {
		return fmt.Errorf("点击编辑器失败: %w", err)
	}
	time.Sleep(500 * time.Millisecond)

	// 清空并输入标题
	page.Keyboard().Press("Control+KeyA")
	page.Keyboard().Press("Delete")
	page.Keyboard().Type(title)
	page.Keyboard().Press("Enter")

	// 输入描述（如果有）
	if description != "" {
		page.Keyboard().Press("Enter")
		page.Keyboard().Type(description)
	}

	utils.InfoWithPlatform(u.platform, "标题和描述已填写")
	time.Sleep(500 * time.Millisecond)
	return nil
}

// addTags 添加话题标签
func (u *Uploader) addTags(page playwright.Page, tags []string) error {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("添加%d个标签...", len(tags)))

	for _, tag := range tags {
		cleanTag := strings.TrimSpace(tag)
		cleanTag = strings.ReplaceAll(cleanTag, "#", "")
		if cleanTag == "" {
			continue
		}

		page.Keyboard().Type("#" + cleanTag)
		page.Keyboard().Press("Space")
		time.Sleep(500 * time.Millisecond)
	}

	utils.InfoWithPlatform(u.platform, "标签添加完成")
	return nil
}

// setOriginal 设置原创声明（Python版完整逻辑）
func (u *Uploader) setOriginal(page playwright.Page, category string) error {
	utils.InfoWithPlatform(u.platform, "设置原创声明...")

	// 步骤1：勾选"视频为原创"
	originalCheckbox := page.GetByLabel("视频为原创").First()
	if count, _ := originalCheckbox.Count(); count > 0 {
		if err := originalCheckbox.Check(); err != nil {
			return fmt.Errorf("勾选原创声明失败: %w", err)
		}
	}

	// 步骤2：检查并勾选"我已阅读并同意《视频号原创声明使用条款》"
	agreementLabel := page.Locator("label:has-text('我已阅读并同意 《视频号原创声明使用条款》')").First()
	isVisible, _ := agreementLabel.IsVisible()
	if isVisible {
		agreementCheckbox := page.GetByLabel("我已阅读并同意 《视频号原创声明使用条款》").First()
		if err := agreementCheckbox.Check(); err != nil {
			return fmt.Errorf("勾选使用条款失败: %w", err)
		}

		// 点击"声明原创"按钮
		declareBtn := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "声明原创"}).First()
		if err := declareBtn.Click(); err != nil {
			return fmt.Errorf("点击声明原创按钮失败: %w", err)
		}
	}

	// 步骤3：处理新版原创声明页面（2023年11月20日更新）
	newOriginalLabel := page.Locator("div.label span:has-text('声明原创')").First()
	if count, _ := newOriginalLabel.Count(); count > 0 && category != "" {
		// 检测原创复选框是否可用（因处罚可能无法勾选）
		originalCheckboxNew := page.Locator("div.declare-original-checkbox input.ant-checkbox-input").First()
		isDisabled, _ := originalCheckboxNew.IsDisabled()

		if !isDisabled {
			if err := originalCheckboxNew.Click(); err != nil {
				return fmt.Errorf("点击新版原创复选框失败: %w", err)
			}

			// 检查是否已勾选
			checkedWrapper := page.Locator("div.declare-original-dialog label.ant-checkbox-wrapper.ant-checkbox-wrapper-checked:visible").First()
			if count, _ := checkedWrapper.Count(); count == 0 {
				// 未勾选，再次点击
				visibleCheckbox := page.Locator("div.declare-original-dialog input.ant-checkbox-input:visible").First()
				visibleCheckbox.Click()
			}
		}

		// 选择原创类型
		originalTypeForm := page.Locator("div.original-type-form > div.form-label:has-text('原创类型'):visible").First()
		if count, _ := originalTypeForm.Count(); count > 0 {
			// 点击下拉菜单
			dropdown := page.Locator("div.form-content:visible").First()
			if err := dropdown.Click(); err != nil {
				return fmt.Errorf("点击原创类型下拉菜单失败: %w", err)
			}
			time.Sleep(1 * time.Second)

			// 选择指定类型
			typeOption := page.Locator(fmt.Sprintf("div.form-content:visible ul.weui-desktop-dropdown__list li.weui-desktop-dropdown__list-ele:has-text('%s')", category)).First()
			if err := typeOption.Click(); err != nil {
				return fmt.Errorf("选择原创类型失败: %w", err)
			}
			time.Sleep(1 * time.Second)
		}

		// 点击声明原创按钮
		declareBtnNew := page.Locator("button:has-text('声明原创'):visible").First()
		if count, _ := declareBtnNew.Count(); count > 0 {
			if err := declareBtnNew.Click(); err != nil {
				return fmt.Errorf("点击新版声明原创按钮失败: %w", err)
			}
		}
	}

	utils.InfoWithPlatform(u.platform, "原创声明已设置")
	time.Sleep(500 * time.Millisecond)
	return nil
}

// addToCollection 添加到合集（Python版逻辑）
func (u *Uploader) addToCollection(page playwright.Page, collection string) error {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("添加到合集: %s", collection))

	// 基于DOM截图优化：使用form-item结构定位"添加到合集"
	// DOM结构: div.form-item > div.form-label:contains("添加到合集") + div.form-content
	collectionSection := page.Locator("div.form-item:has(div.form-label:has-text('添加到合集'))")

	// 检查合集区域是否存在
	if count, _ := collectionSection.Count(); count == 0 {
		utils.InfoWithPlatform(u.platform, "未找到合集区域，跳过添加")
		return nil
	}

	// 获取合集中间区域
	collectionContent := collectionSection.Locator("div.form-content").First()

	// Python版：先检测合集数量>1才点击
	collectionElements := collectionContent.Locator(".option-list-wrap > div")
	count, err := collectionElements.Count()
	if err != nil {
		return fmt.Errorf("获取合集列表失败: %w", err)
	}

	// 只有当合集数量大于1时才点击添加
	if count > 1 {
		if err := collectionContent.Click(); err != nil {
			return fmt.Errorf("点击合集选项失败: %w", err)
		}
		time.Sleep(1 * time.Second)

		// 选择第一个合集（Python版逻辑）
		if err := collectionElements.First().Click(); err != nil {
			return fmt.Errorf("选择合集失败: %w", err)
		}
		time.Sleep(500 * time.Millisecond)

		utils.InfoWithPlatform(u.platform, "已添加到合集")
	} else {
		utils.InfoWithPlatform(u.platform, "合集数量不足，跳过添加")
	}

	return nil
}

// setCover 设置封面
func (u *Uploader) setCover(page playwright.Page, coverPath string) error {
	if _, err := os.Stat(coverPath); err != nil {
		return fmt.Errorf("封面文件不存在: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "设置封面...")

	// 点击封面设置按钮
	coverBtn := page.GetByText("设置封面").First()
	if err := coverBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("未找到封面设置按钮: %w", err)
	}

	if err := coverBtn.Click(); err != nil {
		return fmt.Errorf("点击封面设置按钮失败: %w", err)
	}
	time.Sleep(2 * time.Second)

	// 上传封面
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

	// 点击完成按钮
	finishBtn := page.Locator("button:has-text('完成'), button:has-text('确定')").First()
	if count, _ := finishBtn.Count(); count > 0 {
		if err := finishBtn.Click(); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("点击完成按钮失败: %v", err))
		}
	}
	time.Sleep(1 * time.Second)

	utils.InfoWithPlatform(u.platform, "封面设置完成")
	return nil
}

// formatStrForShortTitle 格式化短标题（Python版逻辑）
// 限制：6-16字符，过滤特殊字符，不足6字符用空格填充
func formatStrForShortTitle(originTitle string) string {
	// 定义允许的特殊字符
	allowedSpecialChars := "《》" + "：+?%°"

	// 移除不允许的特殊字符
	var filteredChars []rune
	for _, char := range originTitle {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
			filteredChars = append(filteredChars, char)
		} else if char == ',' {
			filteredChars = append(filteredChars, ' ')
		} else {
			// 检查是否在允许的特殊字符中
			allowed := false
			for _, allowedChar := range allowedSpecialChars {
				if char == allowedChar {
					allowed = true
					break
				}
			}
			if allowed {
				filteredChars = append(filteredChars, char)
			}
		}
	}
	formattedString := string(filteredChars)

	// 调整字符串长度
	if len([]rune(formattedString)) > 16 {
		// 截断字符串
		runes := []rune(formattedString)
		formattedString = string(runes[:16])
	} else if len([]rune(formattedString)) < 6 {
		// 使用空格来填充字符串
		spacesNeeded := 6 - len([]rune(formattedString))
		formattedString += strings.Repeat(" ", spacesNeeded)
	}

	return formattedString
}

// setShortTitle 设置短标题（Python版逻辑）
func (u *Uploader) setShortTitle(page playwright.Page, title string) error {
	utils.InfoWithPlatform(u.platform, "设置短标题...")

	// 使用Python版的格式化逻辑
	shortTitle := formatStrForShortTitle(title)

	// 基于DOM截图优化：使用input[placeholder*="字数建议6-16个字符"]定位短标题输入框
	shortTitleInput := page.Locator("input[placeholder*='字数建议6-16个字符']").First()

	if count, _ := shortTitleInput.Count(); count > 0 {
		if err := shortTitleInput.Fill(shortTitle); err != nil {
			return fmt.Errorf("填写短标题失败: %w", err)
		}
	}

	utils.InfoWithPlatform(u.platform, fmt.Sprintf("短标题已设置: %s", shortTitle))
	time.Sleep(500 * time.Millisecond)
	return nil
}

// setScheduleTime 设置定时发布（Python版完整日历选择逻辑）
func (u *Uploader) setScheduleTime(page playwright.Page, scheduleTime string) error {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("设置定时发布时间: %s", scheduleTime))

	// 解析时间
	targetTime, err := time.Parse("2006-01-02 15:04", scheduleTime)
	if err != nil {
		return fmt.Errorf("解析时间失败: %w", err)
	}

	// 点击定时发布选项（第2个"定时"标签）
	scheduleLabel := page.Locator("label").Filter(playwright.LocatorFilterOptions{HasText: playwright.String("定时")}).Nth(1)
	if err := scheduleLabel.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(3000),
	}); err != nil {
		return fmt.Errorf("未找到定时发表选项: %w", err)
	}

	if err := scheduleLabel.Click(); err != nil {
		return fmt.Errorf("点击定时发表失败: %w", err)
	}
	time.Sleep(1 * time.Second)

	// 点击时间输入框打开日历（基于DOM截图优化：使用weui-desktop-form-input__extra）
	timeInput := page.Locator("input.weui-desktop-form-input__input[placeholder='请选择发表时间']").First()
	if err := timeInput.Click(); err != nil {
		return fmt.Errorf("点击时间输入框失败: %w", err)
	}
	time.Sleep(1 * time.Second)

	// 获取目标月份
	strMonth := fmt.Sprintf("%02d", targetTime.Month())
	currentMonth := strMonth + "月"

	// 获取页面当前显示的月份
	pageMonth, err := page.InnerText("span.weui-desktop-picker__panel__label:has-text('月')")
	if err != nil {
		return fmt.Errorf("获取当前月份失败: %w", err)
	}

	// 如果月份不匹配，点击下个月按钮
	if pageMonth != currentMonth {
		nextMonthBtn := page.Locator("button.weui-desktop-btn__icon__right").First()
		if err := nextMonthBtn.Click(); err != nil {
			return fmt.Errorf("点击下个月按钮失败: %w", err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	// 获取所有日期元素并选择目标日期
	elements, err := page.QuerySelectorAll("table.weui-desktop-picker__table a")
	if err != nil {
		return fmt.Errorf("获取日期元素失败: %w", err)
	}

	for _, element := range elements {
		// 检查是否禁用
		className, _ := element.Evaluate("el => el.className")
		if className != nil && strings.Contains(className.(string), "weui-desktop-picker__disabled") {
			continue
		}

		// 获取日期文本
		text, err := element.InnerText()
		if err != nil {
			continue
		}

		// 匹配目标日期
		if strings.TrimSpace(text) == fmt.Sprintf("%d", targetTime.Day()) {
			if err := element.Click(); err != nil {
				return fmt.Errorf("点击日期失败: %w", err)
			}
			break
		}
	}

	// 选择时间（小时）
	hourInput := page.Locator("input.weui-desktop-form-input__input[placeholder='请选择时间']").First()
	if err := hourInput.Click(); err != nil {
		return fmt.Errorf("点击时间选择框失败: %w", err)
	}
	time.Sleep(500 * time.Millisecond)

	// 清空并输入小时
	page.Keyboard().Press("Control+KeyA")
	page.Keyboard().Type(fmt.Sprintf("%d", targetTime.Hour()))

	// 点击标题栏让时间生效
	page.Locator("[contenteditable][data-placeholder='添加描述']").Click()

	utils.InfoWithPlatform(u.platform, fmt.Sprintf("定时发布时间设置完成: %s", targetTime.Format("2006-01-02 15:04")))
	time.Sleep(1 * time.Second)
	return nil
}

// publish 点击发布并检测结果（Python版逻辑）
func (u *Uploader) publish(page playwright.Page, browserCtx *browser.PooledContext) error {
	utils.InfoWithPlatform(u.platform, "准备发表...")

	// 基于DOM截图优化：使用button.weui-desktop-btn_primary:has-text('发表')定位发表按钮
	publishBtn := page.Locator("button.weui-desktop-btn_primary:has-text('发表')").First()
	if err := publishBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("未找到发表按钮: %w", err)
	}

	if err := publishBtn.Click(); err != nil {
		return fmt.Errorf("点击发表按钮失败: %w", err)
	}

	// 检测发布结果（Python版while循环+异常捕获逻辑）
	publishTimeout := 30 * time.Second
	publishStart := time.Now()

	for time.Since(publishStart) < publishTimeout {
		if browserCtx.IsPageClosed() {
			return fmt.Errorf("浏览器已关闭")
		}

		// 检测URL跳转（支持通配符匹配）
		url := page.URL()
		if strings.Contains(url, "post/list") {
			utils.InfoWithPlatform(u.platform, "视频发布成功")
			return nil
		}

		// 检测成功提示
		successText := page.Locator("text=/发表成功|发布成功/").First()
		if count, _ := successText.Count(); count > 0 {
			if visible, _ := successText.IsVisible(); visible {
				utils.InfoWithPlatform(u.platform, "发表成功")
				return nil
			}
		}

		utils.InfoWithPlatform(u.platform, "视频正在发布中...")
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("发表超时")
}

// saveDraft 保存草稿（Python版逻辑）
func (u *Uploader) saveDraft(page playwright.Page, browserCtx *browser.PooledContext) error {
	utils.InfoWithPlatform(u.platform, "保存草稿...")

	draftBtn := page.Locator("div.form-btns button:has-text('保存草稿')").First()
	if err := draftBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("未找到保存草稿按钮: %w", err)
	}

	if count, _ := draftBtn.Count(); count > 0 {
		if err := draftBtn.Click(); err != nil {
			return fmt.Errorf("点击保存草稿按钮失败: %w", err)
		}
	}

	// 检测保存结果（Python版while循环+URL检测逻辑）
	draftTimeout := 30 * time.Second
	draftStart := time.Now()

	for time.Since(draftStart) < draftTimeout {
		if browserCtx.IsPageClosed() {
			return fmt.Errorf("浏览器已关闭")
		}

		// 检测URL跳转（支持通配符匹配和draft关键词）
		url := page.URL()
		if strings.Contains(url, "post/list") || strings.Contains(url, "draft") {
			utils.InfoWithPlatform(u.platform, "视频草稿保存成功")
			return nil
		}

		utils.InfoWithPlatform(u.platform, "正在保存草稿...")
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("保存草稿超时")
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

	utils.InfoWithPlatform(u.platform, "正在打开登录页面...")
	if _, err := page.Goto("https://channels.weixin.qq.com/platform/post/create", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		return fmt.Errorf("打开登录页面失败: %w", err)
	}

	time.Sleep(3 * time.Second)

	utils.InfoWithPlatform(u.platform, "请在浏览器窗口中完成登录...")

	// 使用Cookie检测机制等待登录成功
	cookieConfig, ok := browser.GetCookieConfig("tencent")
	if !ok {
		return fmt.Errorf("获取视频号Cookie配置失败")
	}

	if err := browserCtx.WaitForLoginCookies(cookieConfig); err != nil {
		return fmt.Errorf("等待登录Cookie失败: %w", err)
	}

	utils.SuccessWithPlatform(u.platform, "登录成功，检测到wxsid和wxuin Cookie")
	if err := browserCtx.SaveCookiesTo(u.cookiePath); err != nil {
		return fmt.Errorf("保存Cookie失败: %w", err)
	}
	utils.InfoWithPlatform(u.platform, "Cookie已保存")
	return nil
}
