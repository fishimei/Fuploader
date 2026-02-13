package bilibili

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
		platform:    "bilibili",
		browserPool: browser.GetDefaultPool(),
		config:      DefaultConfig(),
	}
	debugLog("创建上传器 - 地址: %p, cookiePath: '%s'", u, cookiePath)
	if cookiePath == "" {
		utils.Warn("[Bilibili] NewUploader 收到空的cookiePath!")
	}
	return u
}

func NewUploaderWithAccount(accountID uint) *Uploader {
	cookiePath := config.GetCookiePath("bilibili", int(accountID))
	u := &Uploader{
		accountID:   accountID,
		cookiePath:  cookiePath,
		platform:    "bilibili",
		browserPool: browser.GetDefaultPool(),
		config:      DefaultConfig(),
	}
	debugLog("创建上传器 - 地址: %p, accountID: %d, cookiePath: '%s'", u, accountID, cookiePath)
	return u
}

func NewUploaderWithPool(accountID uint, pool *browser.Pool) *Uploader {
	cookiePath := config.GetCookiePath("bilibili", int(accountID))
	u := &Uploader{
		accountID:   accountID,
		cookiePath:  cookiePath,
		platform:    "bilibili",
		browserPool: pool,
		config:      DefaultConfig(),
	}
	return u
}

func (u *Uploader) Platform() string {
	return u.platform
}

func (u *Uploader) Upload(ctx context.Context, task *types.VideoTask) error {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("开始上传: %s", task.VideoPath))

	if _, err := os.Stat(task.VideoPath); err != nil {
		return fmt.Errorf("失败: 开始上传 - 视频文件不存在: %w", err)
	}

	browserCtx, err := u.browserPool.GetContextByAccount(ctx, u.accountID, u.cookiePath, nil)
	if err != nil {
		return fmt.Errorf("失败: 开始上传 - 获取浏览器失败: %w", err)
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		return fmt.Errorf("失败: 开始上传 - 获取页面失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "正在打开发布页面...")
	if _, err := page.Goto("https://member.bilibili.com/platform/upload/video/frame", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		return fmt.Errorf("失败: 开始上传 - 打开页面失败: %w", err)
	}

	if err := page.Locator(`input[type="file"]`).First().WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		utils.WarnWithPlatform(u.platform, "等待文件输入框超时")
	}

	utils.InfoWithPlatform(u.platform, "正在上传视频...")

	fileInput := page.Locator(`div.bcc-upload-wrapper input[type="file"][accept*=".mp4"][style*="display: none"]`).First()
	count, _ := fileInput.Count()

	if count == 0 {
		fileInput = page.Locator(`div.bcc-upload-wrapper input[type="file"]`).First()
		count, _ = fileInput.Count()
	}
	if count == 0 {
		fileInput = page.Locator(`input[type="file"][accept*=".mp4"]`).First()
		count, _ = fileInput.Count()
	}
	if count == 0 {
		fileInput = page.Locator(`input[type="file"]`).First()
		count, _ = fileInput.Count()
	}

	if count == 0 {
		return fmt.Errorf("未找到文件上传输入框")
	}

	utils.InfoWithPlatform(u.platform, fmt.Sprintf("找到文件输入框，count=%d", count))

	if err := fileInput.SetInputFiles(task.VideoPath); err != nil {
		return fmt.Errorf("失败: 选择视频文件 - %w", err)
	}

	utils.InfoWithPlatform(u.platform, "等待视频上传完成...")

	if err := u.waitForUploadComplete(ctx, page, browserCtx); err != nil {
		return err
	}

	if err := page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateDomcontentloaded,
	}); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("等待页面加载超时: %v", err))
	}

	if task.Copyright != "" {
		u.setCopyright(page, task.Copyright)
	}

	if task.Title != "" {
		u.setTitle(page, task.Title)
	}

	u.setTags(page, task.Tags)

	if task.Description != "" {
		u.setDescription(page, task.Description)
	}

	utils.InfoWithPlatform(u.platform, "设置封面...")
	coverFilled, err := u.setCover(page, task.Thumbnail)
	if err != nil {
		utils.WarnWithPlatform(u.platform, err.Error())
	} else if coverFilled {
		utils.InfoWithPlatform(u.platform, "封面设置完成")
	} else {
		utils.WarnWithPlatform(u.platform, "封面设置可能未完成")
	}

	if task.ScheduleTime != nil && *task.ScheduleTime != "" {
		if err := u.setScheduleTime(page, *task.ScheduleTime); err != nil {
			utils.WarnWithPlatform(u.platform, err.Error())
		}
	}

	return u.submitVideo(ctx, page, browserCtx)
}

func (u *Uploader) waitForUploadComplete(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext) error {
	uploadStartTime := time.Now()
	uploadCompleted := false
	lastProgressCount := -1
	stuckCount := 0
	checkInterval := 2 * time.Second

	for time.Since(uploadStartTime) < u.config.UploadTimeout {
		select {
		case <-ctx.Done():
			return fmt.Errorf("上传已取消")
		default:
		}

		if browserCtx.IsPageClosed() {
			return fmt.Errorf("浏览器已关闭")
		}

		progressBar := page.Locator(`.bcc-upload-progress, .upload-progress, [class*="progress"]`).First()
		uploadSuccessText := page.Locator(`text=/上传完成|转码中|处理中|视频上传成功/`).First()
		uploadDoneIcon := page.Locator(`.upload-done, .upload-success, [class*="success"]`).First()

		progressCount, _ := progressBar.Count()
		successCount, _ := uploadSuccessText.Count()
		doneCount, _ := uploadDoneIcon.Count()

		if config.Config != nil && config.Config.DebugMode {
			utils.InfoWithPlatform(u.platform, fmt.Sprintf("[调试] 进度条: %d, 成功文本: %d, 完成图标: %d", progressCount, successCount, doneCount))
		}

		if (progressCount == 0 && successCount > 0) || doneCount > 0 {
			utils.SuccessWithPlatform(u.platform, "视频上传完成")
			uploadCompleted = true
			break
		}

		if progressCount == lastProgressCount {
			stuckCount++
			if progressCount == 0 && stuckCount >= 3 {
				utils.SuccessWithPlatform(u.platform, "视频上传完成")
				uploadCompleted = true
				break
			}
		} else {
			stuckCount = 0
		}
		lastProgressCount = progressCount

		uploadError := page.Locator(`text=/上传失败|错误|失败|Upload failed/`).First()
		if count, _ := uploadError.Count(); count > 0 {
			return fmt.Errorf("失败: 视频上传 - 上传失败")
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("上传已取消")
		case <-time.After(checkInterval):
		}
	}

	if !uploadCompleted {
		return fmt.Errorf("失败: 视频上传 - 超时")
	}

	return nil
}

func (u *Uploader) setCopyright(page playwright.Page, copyright string) {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("设置转载类型: %s", copyright))
	var copyrightText string
	if copyright == "1" {
		copyrightText = "自制"
	} else if copyright == "2" {
		copyrightText = "转载"
	}
	if copyrightText == "" {
		return
	}

	copyrightLocator := page.Locator(fmt.Sprintf(`span:has-text("%s")`, copyrightText)).First()
	if err := copyrightLocator.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds()))}); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("等待%s选项超时: %v", copyrightText, err))
		return
	}

	if count, _ := copyrightLocator.Count(); count > 0 {
		if err := copyrightLocator.Click(); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("点击%s选项失败: %v", copyrightText, err))
		} else {
			utils.InfoWithPlatform(u.platform, fmt.Sprintf("已选择%s", copyrightText))
		}
	}
}

func (u *Uploader) setTitle(page playwright.Page, title string) {
	utils.InfoWithPlatform(u.platform, "填写标题...")
	titleInput := page.Locator(`input[type="text"][placeholder="请输入稿件标题"]`).First()
	if err := titleInput.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds()))}); err != nil {
		titleInput = page.Locator(`div.video-title-container input[type="text"]`).First()
	}
	if count, _ := titleInput.Count(); count > 0 {
		if err := titleInput.Fill(title); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("填写标题失败: %v", err))
		} else {
			utils.InfoWithPlatform(u.platform, fmt.Sprintf("标题已填写: %s", title))
		}
	}
}

func (u *Uploader) setTags(page playwright.Page, tags []string) {
	utils.InfoWithPlatform(u.platform, "添加标签...")

	for {
		tagCloseBtn := page.Locator(`svg.close.icon-sprite.icon-sprite-off`).First()
		if count, _ := tagCloseBtn.Count(); count == 0 {
			tagCloseBtn = page.Locator(`div.label-item-v2-container >> svg.close`).First()
		}
		if count, _ := tagCloseBtn.Count(); count == 0 {
			break
		}
		if err := tagCloseBtn.Click(); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("删除默认标签失败: %v", err))
			break
		}
	}

	if len(tags) == 0 {
		return
	}

	tagInput := page.Locator(`div.tag-input-wrp >> input[type="text"]`).First()
	if err := tagInput.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds()))}); err != nil {
		tagInput = page.Locator(`input[type="text"][placeholder="按回车键Enter创建标签"]`).First()
	}

	if count, _ := tagInput.Count(); count == 0 {
		utils.WarnWithPlatform(u.platform, "未找到标签输入框")
		return
	}

	for i, tag := range tags {
		if err := tagInput.Fill(tag); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("输入标签[%d]失败: %v", i, err))
			continue
		}
		if err := tagInput.Press("Enter"); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("确认标签[%d]失败: %v", i, err))
		}
	}
	utils.InfoWithPlatform(u.platform, "标签添加完成")
}

func (u *Uploader) setDescription(page playwright.Page, description string) {
	utils.InfoWithPlatform(u.platform, "填写描述...")
	descEditor := page.Locator(`div.ql-editor[data-placeholder*="相关信息"]`).First()
	if err := descEditor.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds()))}); err != nil {
		descEditor = page.Locator(`div.desc-text-wrp div.ql-editor`).First()
	}
	if count, _ := descEditor.Count(); count == 0 {
		descEditor = page.Locator(`div.archive-info-editor div.ql-editor`).First()
	}
	if count, _ := descEditor.Count(); count > 0 {
		if err := descEditor.Fill(description); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("填写描述失败: %v", err))
		} else {
			utils.InfoWithPlatform(u.platform, "描述已填写")
		}
	}
}

func (u *Uploader) submitVideo(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext) error {
	utils.InfoWithPlatform(u.platform, "准备发布...")

	submitBtn := page.Locator(`span.submit-add:text("立即投稿")`).First()
	if err := submitBtn.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds()))}); err != nil {
		submitBtn = page.Locator(`span[data-reporter-id="89"].submit-add`).First()
		if err := submitBtn.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds()))}); err != nil {
			submitBtn = page.Locator(`div.submit-container >> span.submit-add`).First()
			if err := submitBtn.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds()))}); err != nil {
				utils.WarnWithPlatform(u.platform, fmt.Sprintf("等待发布按钮超时: %v", err))
			}
		}
	}

	if count, _ := submitBtn.Count(); count == 0 {
		return fmt.Errorf("未找到发布按钮")
	}

	if err := submitBtn.ScrollIntoViewIfNeeded(); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("滚动到发布按钮失败: %v", err))
	}

	urlBeforeSubmit := page.URL()
	submitSuccess := false

	for clickAttempt := 1; clickAttempt <= u.config.MaxClickAttempts && !submitSuccess; clickAttempt++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("发布已取消")
		default:
		}

		utils.InfoWithPlatform(u.platform, fmt.Sprintf("第%d次尝试发布...", clickAttempt))

		if err := submitBtn.Click(playwright.LocatorClickOptions{Force: playwright.Bool(true)}); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("点击发布按钮失败: %v", err))
			time.Sleep(2 * time.Second)
			continue
		}

		time.Sleep(3 * time.Second)

		confirmDialogBtn := page.Locator(`button:has-text("确定"), button:has-text("确认")`).First()
		if count, _ := confirmDialogBtn.Count(); count > 0 {
			confirmDialogBtn.Click()
			time.Sleep(2 * time.Second)
		}

		submitSuccess = u.checkSubmitSuccess(ctx, page, browserCtx, urlBeforeSubmit, submitBtn)

		if !submitSuccess && clickAttempt < u.config.MaxClickAttempts {
			time.Sleep(3 * time.Second)
		}
	}

	if !submitSuccess {
		return fmt.Errorf("发布失败")
	}

	utils.SuccessWithPlatform(u.platform, "发布成功")
	return nil
}

func (u *Uploader) checkSubmitSuccess(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext, urlBeforeSubmit string, submitBtn playwright.Locator) bool {
	submitCheckStart := time.Now()
	checkInterval := 2 * time.Second

	for time.Since(submitCheckStart) < u.config.SubmitCheckTimeout {
		select {
		case <-ctx.Done():
			return false
		default:
		}

		if browserCtx.IsPageClosed() {
			return false
		}

		currentURL := page.URL()

		if strings.Contains(currentURL, "member.bilibili.com/platform/upload/manage") ||
			strings.Contains(currentURL, "member.bilibili.com/platform/home") {
			return true
		}

		if currentURL != urlBeforeSubmit && !strings.Contains(currentURL, "frame") {
			return true
		}

		successToast := page.Locator(`text=/投稿成功|发布成功|提交成功|审核中|稿件已提交/`).First()
		if count, _ := successToast.Count(); count > 0 {
			text, _ := successToast.TextContent()
			if !strings.Contains(text, "投稿中") && !strings.Contains(text, "处理中") && !strings.Contains(text, "正在提交") {
				return true
			}
		}

		submitBtnCount, _ := submitBtn.Count()
		if submitBtnCount == 0 {
			time.Sleep(2 * time.Second)
			if count, _ := submitBtn.Count(); count == 0 {
				return true
			}
		}

		errorToast := page.Locator(`text=/投稿失败|发布失败|提交失败|错误|请完善/`).First()
		if count, _ := errorToast.Count(); count > 0 {
			text, _ := errorToast.TextContent()
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("发布失败: %s", text))
			return false
		}

		select {
		case <-ctx.Done():
			return false
		case <-time.After(checkInterval):
		}
	}

	return false
}
