package baijiahao

import (
	"context"
	"fmt"
	"math/rand"
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
		utils.InfoWithPlatform("baijiahao", fmt.Sprintf("[调试] "+format, args...))
	}
}

// browserPool 全局浏览器池实例
var browserPool *browser.Pool

func init() {
	browserPool = browser.NewPool(2, 5)
}

// Uploader 百家号上传器
type Uploader struct {
	cookiePath string
	platform   string
}

// NewUploader 创建上传器
func NewUploader(cookiePath string) *Uploader {
	u := &Uploader{
		cookiePath: cookiePath,
		platform:   "baijiahao",
	}
	debugLog("创建上传器 - 地址: %p, cookiePath: '%s'", u, cookiePath)
	if cookiePath == "" {
		utils.Warn("[Baijiahao] NewUploader 收到空的cookiePath!")
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

	if _, err := page.Goto("https://baijiahao.baidu.com/builder/rc/home", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("打开页面失败: %v", err))
		return false, nil
	}

	time.Sleep(5 * time.Second)

	// 使用Cookie检测机制验证登录状态
	cookieConfig, ok := browser.GetCookieConfig("baijiahao")
	if !ok {
		return false, fmt.Errorf("获取百家号Cookie配置失败")
	}

	isValid, err := browserCtx.ValidateLoginCookies(cookieConfig)
	if err != nil {
		return false, fmt.Errorf("验证Cookie失败: %w", err)
	}

	if isValid {
		utils.InfoWithPlatform(u.platform, "检测到PTOKEN和BAIDUID Cookie，验证通过")
	} else {
		utils.InfoWithPlatform(u.platform, "未检测到PTOKEN和BAIDUID Cookie，验证失败")
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
	if _, err := page.Goto("https://baijiahao.baidu.com/builder/rc/edit?type=videoV2", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		return fmt.Errorf("打开发布页面失败: %w", err)
	}

	// 等待页面加载
	utils.InfoWithPlatform(u.platform, "等待页面加载...")
	if err := page.Locator("div#formMain:visible").WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(10000),
	}); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("等待页面加载超时: %v", err))
	}
	time.Sleep(3 * time.Second)

	// 上传视频
	utils.InfoWithPlatform(u.platform, "正在上传视频...")
	inputLocator := page.Locator("div[class^='video-main-container'] input[type='file']").First()
	if count, _ := inputLocator.Count(); count == 0 {
		inputLocator = page.Locator("input[type='file']").First()
	}

	if err := inputLocator.SetInputFiles(task.VideoPath); err != nil {
		return fmt.Errorf("上传视频失败: %w", err)
	}

	// 等待上传完成
	utils.InfoWithPlatform(u.platform, "等待视频上传完成...")
	if err := u.waitForUploadComplete(ctx, page, browserCtx); err != nil {
		return fmt.Errorf("等待上传完成失败: %w", err)
	}

	time.Sleep(2 * time.Second)

	// 填写标题
	if err := u.fillTitle(page, task.Title); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("填写标题失败: %v", err))
	}

	// 填写描述
	if task.Description != "" {
		if err := u.fillDescription(page, task.Description); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("填写描述失败: %v", err))
		}
	}

	// 添加标签
	if len(task.Tags) > 0 {
		if err := u.addTags(page, task.Tags); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("添加标签失败: %v", err))
		}
	}

	// 等待封面生成
	utils.InfoWithPlatform(u.platform, "等待封面生成...")
	if err := u.waitForCoverGenerated(page); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("等待封面生成失败: %v", err))
	}

	// 设置自定义封面
	if task.Thumbnail != "" {
		if err := u.setCustomCover(page, task.Thumbnail); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("设置自定义封面失败: %v", err))
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

// waitForUploadComplete 等待视频上传完成
func (u *Uploader) waitForUploadComplete(ctx context.Context, page playwright.Page, browserCtx *browser.Context) error {
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

		// 检测上传成功标志
		// 方式1：检查视频预览区域出现
		videoPreview := page.Locator("div.video-preview, div[class^='videoPreview']").First()
		if count, _ := videoPreview.Count(); count > 0 {
			if visible, _ := videoPreview.IsVisible(); visible {
				utils.InfoWithPlatform(u.platform, "视频上传完成（检测到预览区域）")
				return nil
			}
		}

		// 方式2：检查上传进度条消失
		progressBar := page.Locator("div[class^='upload-progress'], div.progress-bar").First()
		if count, _ := progressBar.Count(); count == 0 {
			// 进度条消失，检查是否有视频信息
			videoInfo := page.Locator("div.video-info, div[class^='videoInfo']").First()
			if count, _ := videoInfo.Count(); count > 0 {
				utils.InfoWithPlatform(u.platform, "视频上传完成（进度条消失且检测到视频信息）")
				return nil
			}
		}

		// 方式3：检查"上传成功"文本
		successText := page.Locator("text=/上传成功|上传完成|视频已上传/").First()
		if count, _ := successText.Count(); count > 0 {
			if visible, _ := successText.IsVisible(); visible {
				utils.InfoWithPlatform(u.platform, "视频上传完成（检测到成功文本）")
				return nil
			}
		}

		// 检测上传失败
		errorText := page.Locator("text=/上传失败|上传出错|Upload failed/").First()
		if count, _ := errorText.Count(); count > 0 {
			return fmt.Errorf("视频上传失败")
		}

		time.Sleep(uploadCheckInterval)
	}

	return fmt.Errorf("上传超时")
}

// fillTitle 填写标题
func (u *Uploader) fillTitle(page playwright.Page, title string) error {
	if title == "" {
		return nil
	}

	utils.InfoWithPlatform(u.platform, "填写标题...")

	// 百家号标题少于8字符时自动补全
	if len(title) < 8 {
		title = title + " 你不知道的"
		utils.InfoWithPlatform(u.platform, fmt.Sprintf("标题少于8字符，自动补全为: %s", title))
	}

	// 限制30字符
	if len(title) > 30 {
		runes := []rune(title)
		if len(runes) > 30 {
			title = string(runes[:30])
		}
	}

	titleInput := page.GetByPlaceholder("添加标题获得更多推荐")
	if err := titleInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		// 兜底：尝试其他选择器
		titleInput = page.Locator("input[placeholder*='标题']").First()
	}

	if count, _ := titleInput.Count(); count == 0 {
		return fmt.Errorf("未找到标题输入框")
	}

	if err := titleInput.Fill(title); err != nil {
		return fmt.Errorf("填写标题失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, fmt.Sprintf("标题已填写: %s", title))
	time.Sleep(500 * time.Millisecond)
	return nil
}

// fillDescription 填写描述
func (u *Uploader) fillDescription(page playwright.Page, description string) error {
	utils.InfoWithPlatform(u.platform, "填写描述...")

	// 尝试多种选择器定位描述输入框
	descInput := page.Locator("textarea[placeholder*='简介'], textarea[placeholder*='描述'], div.ql-editor[data-placeholder*='简介']").First()
	if err := descInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(3000),
	}); err != nil {
		// 兜底：尝试富文本编辑器
		descInput = page.Locator("div.ql-editor").First()
	}

	if count, _ := descInput.Count(); count == 0 {
		return fmt.Errorf("未找到描述输入框")
	}

	if err := descInput.Fill(description); err != nil {
		return fmt.Errorf("填写描述失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "描述已填写")
	time.Sleep(500 * time.Millisecond)
	return nil
}

// addTags 添加标签
func (u *Uploader) addTags(page playwright.Page, tags []string) error {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("添加%d个标签...", len(tags)))

	// 定位标签输入框
	tagInput := page.Locator("input[placeholder*='标签'], input[placeholder*='话题']").First()
	if err := tagInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(3000),
	}); err != nil {
		return fmt.Errorf("未找到标签输入框: %w", err)
	}

	for i, tag := range tags {
		// 清理标签中的特殊字符
		cleanTag := strings.TrimSpace(tag)
		cleanTag = strings.ReplaceAll(cleanTag, "#", "")
		cleanTag = strings.ReplaceAll(cleanTag, " ", "")

		if cleanTag == "" {
			continue
		}

		if err := tagInput.Fill(cleanTag); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("输入标签[%d]失败: %v", i, err))
			continue
		}
		time.Sleep(300 * time.Millisecond)

		// 按Enter确认标签
		if err := tagInput.Press("Enter"); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("确认标签[%d]失败: %v", i, err))
			continue
		}
		time.Sleep(500 * time.Millisecond)
	}

	utils.InfoWithPlatform(u.platform, "标签添加完成")
	return nil
}

// waitForCoverGenerated 等待封面自动生成
func (u *Uploader) waitForCoverGenerated(page playwright.Page) error {
	coverTimeout := 30 * time.Second
	coverCheckInterval := 1 * time.Second
	coverStartTime := time.Now()

	for time.Since(coverStartTime) < coverTimeout {
		// 检查封面图片是否加载完成
		coverImg := page.Locator("div.cheetah-spin-container img, div[class^='cover'] img, .cover-preview img").First()
		if count, _ := coverImg.Count(); count > 0 {
			if visible, _ := coverImg.IsVisible(); visible {
				utils.InfoWithPlatform(u.platform, "封面已生成")
				return nil
			}
		}

		// 检查封面区域是否有内容
		coverArea := page.Locator("div[class^='cover'], div.cover-area").First()
		if count, _ := coverArea.Count(); count > 0 {
			// 检查是否有背景图或子元素
			hasContent, _ := coverArea.Evaluate(`el => {
				return el.querySelector('img') !== null || 
				       el.style.backgroundImage !== '' && el.style.backgroundImage !== 'none'
			}`)
			if hasContent.(bool) {
				utils.InfoWithPlatform(u.platform, "封面已生成")
				return nil
			}
		}

		time.Sleep(coverCheckInterval)
	}

	return fmt.Errorf("等待封面生成超时")
}

// setCustomCover 设置自定义封面
func (u *Uploader) setCustomCover(page playwright.Page, coverPath string) error {
	if _, err := os.Stat(coverPath); err != nil {
		return fmt.Errorf("封面文件不存在: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "设置自定义封面...")

	// 点击封面区域打开选择弹窗
	coverArea := page.Locator("div[class^='cover'], div.cover-area, div.cheetah-spin-container").First()
	if err := coverArea.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("未找到封面区域: %w", err)
	}

	if err := coverArea.Click(); err != nil {
		return fmt.Errorf("点击封面区域失败: %w", err)
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

	// 点击确认或完成按钮
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

	// 百家号定时发布范围：未来2-24小时（Python版使用随机时间）
	now := time.Now()
	minTime := now.Add(2 * time.Hour)
	maxTime := now.Add(24 * time.Hour)

	if targetTime.Before(minTime) {
		// 如果指定时间不符合要求，使用随机时间
		randomMinutes := time.Duration(rand.Intn(22*60)+120) * time.Minute
		targetTime = now.Add(randomMinutes)
		utils.InfoWithPlatform(u.platform, fmt.Sprintf("指定时间不符合要求，使用随机时间: %s", targetTime.Format("2006-01-02 15:04")))
	}
	if targetTime.After(maxTime) {
		targetTime = maxTime
	}

	// 点击定时发布选项
	scheduleLabel := page.Locator("label:has-text('定时发布'), text='定时发布'").First()
	if err := scheduleLabel.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(3000),
	}); err != nil {
		return fmt.Errorf("未找到定时发布选项: %w", err)
	}

	if err := scheduleLabel.Click(); err != nil {
		return fmt.Errorf("点击定时发布失败: %w", err)
	}
	time.Sleep(1 * time.Second)

	// 选择时间
	timeInput := page.Locator("input[placeholder*='时间'], input[placeholder*='日期']").First()
	if err := timeInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(3000),
	}); err != nil {
		return fmt.Errorf("未找到时间输入框: %w", err)
	}

	if err := timeInput.Click(); err != nil {
		return fmt.Errorf("点击时间输入框失败: %w", err)
	}
	time.Sleep(500 * time.Millisecond)

	// 输入时间
	timeStr := targetTime.Format("2006-01-02 15:04")
	if err := timeInput.Fill(timeStr); err != nil {
		return fmt.Errorf("输入时间失败: %w", err)
	}

	// 按Enter确认
	if err := timeInput.Press("Enter"); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("按Enter确认失败: %v", err))
	}

	utils.InfoWithPlatform(u.platform, fmt.Sprintf("定时发布时间设置完成: %s", timeStr))
	time.Sleep(1 * time.Second)
	return nil
}

// publish 点击发布并检测结果
func (u *Uploader) publish(page playwright.Page, browserCtx *browser.Context) error {
	// 定位发布按钮
	publishBtn := page.Locator("button:has-text('发布'), button:has-text('立即发布')").First()
	if err := publishBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("未找到发布按钮: %w", err)
	}

	// 滚动到按钮可见
	if err := publishBtn.ScrollIntoViewIfNeeded(); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("滚动到发布按钮失败: %v", err))
	}

	urlBeforePublish := page.URL()
	maxAttempts := 3

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if browserCtx.IsPageClosed() {
			return fmt.Errorf("浏览器已关闭")
		}

		utils.InfoWithPlatform(u.platform, fmt.Sprintf("第%d次尝试发布...", attempt+1))

		// 点击发布按钮
		if err := publishBtn.Click(playwright.LocatorClickOptions{
			Force: playwright.Bool(true),
		}); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("点击发布按钮失败: %v", err))
			time.Sleep(2 * time.Second)
			continue
		}

		utils.InfoWithPlatform(u.platform, "已点击发布按钮")
		time.Sleep(3 * time.Second)

		// 处理确认弹窗
		confirmDialog := page.Locator("button:has-text('确定'), button:has-text('确认')").First()
		if count, _ := confirmDialog.Count(); count > 0 {
			if visible, _ := confirmDialog.IsVisible(); visible {
				utils.InfoWithPlatform(u.platform, "处理确认弹窗...")
				confirmDialog.Click()
				time.Sleep(2 * time.Second)
			}
		}

		// 检测发布结果
		if err := u.checkPublishResult(page, browserCtx, urlBeforePublish); err == nil {
			return nil
		} else {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("发布检测未通过: %v", err))
		}

		if attempt < maxAttempts-1 {
			time.Sleep(3 * time.Second)
		}
	}

	return fmt.Errorf("发布失败，已重试%d次", maxAttempts)
}

// checkPublishResult 检测发布结果
func (u *Uploader) checkPublishResult(page playwright.Page, browserCtx *browser.Context, urlBefore string) error {
	checkTimeout := 60 * time.Second
	checkInterval := 2 * time.Second
	checkStart := time.Now()

	for time.Since(checkStart) < checkTimeout {
		if browserCtx.IsPageClosed() {
			return fmt.Errorf("浏览器已关闭")
		}

		currentURL := page.URL()

		// 成功标志1：URL跳转到管理页
		if strings.Contains(currentURL, "baijiahao.baidu.com/builder/rc/clue") ||
			strings.Contains(currentURL, "baijiahao.baidu.com/builder/rc/manage") {
			utils.InfoWithPlatform(u.platform, "发布成功，页面已跳转到管理页")
			return nil
		}

		// 成功标志2：URL变化且不再包含edit
		if currentURL != urlBefore && !strings.Contains(currentURL, "edit") {
			utils.InfoWithPlatform(u.platform, "发布成功，URL已变化")
			return nil
		}

		// 成功标志3：成功提示文本
		successIndicators := []string{"发布成功", "提交成功", "审核中", "稿件已提交"}
		for _, indicator := range successIndicators {
			successText := page.Locator(fmt.Sprintf("text=%s", indicator)).First()
			if count, _ := successText.Count(); count > 0 {
				if visible, _ := successText.IsVisible(); visible {
					text, _ := successText.TextContent()
					utils.InfoWithPlatform(u.platform, fmt.Sprintf("检测到成功提示: %s", text))
					return nil
				}
			}
		}

		// 失败标志
		errorIndicators := []string{"发布失败", "提交失败", "错误", "请完善"}
		for _, indicator := range errorIndicators {
			errorText := page.Locator(fmt.Sprintf("text=%s", indicator)).First()
			if count, _ := errorText.Count(); count > 0 {
				if visible, _ := errorText.IsVisible(); visible {
					text, _ := errorText.TextContent()
					return fmt.Errorf("发布出错: %s", text)
				}
			}
		}

		time.Sleep(checkInterval)
	}

	return fmt.Errorf("发布检测超时")
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
	if _, err := page.Goto("https://baijiahao.baidu.com/builder/theme/bjh/login", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		return fmt.Errorf("打开登录页面失败: %w", err)
	}

	time.Sleep(3 * time.Second)

	utils.InfoWithPlatform(u.platform, "请在浏览器窗口中完成登录...")

	// 使用Cookie检测机制等待登录成功
	cookieConfig, ok := browser.GetCookieConfig("baijiahao")
	if !ok {
		return fmt.Errorf("获取百家号Cookie配置失败")
	}

	if err := browserCtx.WaitForLoginCookies(cookieConfig); err != nil {
		return fmt.Errorf("等待登录Cookie失败: %w", err)
	}

	utils.SuccessWithPlatform(u.platform, "登录成功，检测到PTOKEN和BAIDUID Cookie")
	if err := browserCtx.SaveCookiesTo(u.cookiePath); err != nil {
		return fmt.Errorf("保存Cookie失败: %w", err)
	}
	utils.InfoWithPlatform(u.platform, "Cookie已保存")
	return nil
}
