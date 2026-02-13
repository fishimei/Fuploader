package tencent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"Fuploader/internal/platform/browser"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

var videoFileInputSelectors = []string{
	`span.ant-upload.ant-upload-btn > input[type="file"][accept^="video/"][style*="display: none"]`,
	`input[type='file'][accept*='video']`,
	`input[type='file']`,
}

func (u *Uploader) getVideoFileInput(page playwright.Page) playwright.Locator {
	for _, selector := range videoFileInputSelectors {
		locator := page.Locator(selector).First()
		if count, _ := locator.Count(); count > 0 {
			return locator
		}
	}
	return page.Locator(videoFileInputSelectors[len(videoFileInputSelectors)-1]).First()
}

func (u *Uploader) waitForVideoFileInput(page playwright.Page) (playwright.Locator, error) {
	timeout := float64(u.config.ElementWaitTimeout.Milliseconds())
	for _, selector := range videoFileInputSelectors {
		locator := page.Locator(selector).First()
		if err := locator.WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(timeout),
		}); err == nil {
			return locator, nil
		}
	}
	return nil, fmt.Errorf("未找到视频文件输入框")
}

func (u *Uploader) uploadVideo(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext, videoPath string) error {
	utils.InfoWithPlatform(u.platform, "正在上传视频...")

	fileInput, err := u.waitForVideoFileInput(page)
	if err != nil {
		return fmt.Errorf("失败: 上传视频 - %w", err)
	}

	if err := fileInput.SetInputFiles(videoPath); err != nil {
		return fmt.Errorf("失败: 上传视频 - %w", err)
	}

	utils.InfoWithPlatform(u.platform, "等待视频上传完成...")
	if err := u.waitForUploadComplete(ctx, page, browserCtx, videoPath); err != nil {
		return err
	}

	return nil
}

func (u *Uploader) waitForUploadComplete(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext, videoPath string) error {
	uploadStartTime := time.Now()
	retryCount := 0

	for time.Since(uploadStartTime) < u.config.UploadTimeout {
		select {
		case <-ctx.Done():
			return fmt.Errorf("失败: 等待上传完成 - 上传已取消")
		default:
		}

		if browserCtx.IsPageClosed() {
			return fmt.Errorf("失败: 等待上传完成 - 浏览器已关闭")
		}

		publishBtn := page.Locator("button.weui-desktop-btn_primary:has-text('发表')").First()
		if count, _ := publishBtn.Count(); count > 0 {
			classAttr, _ := publishBtn.GetAttribute("class")
			if classAttr != "" && !strings.Contains(classAttr, "weui-desktop-btn_disabled") {
				utils.InfoWithPlatform(u.platform, "视频上传完成")
				return nil
			}
		}

		errorMsg := page.Locator("div.status-msg.error").First()
		deleteBtn := page.Locator("div.media-status-content div.tag-inner:has-text('删除')").First()
		if errorCount, _ := errorMsg.Count(); errorCount > 0 {
			if deleteCount, _ := deleteBtn.Count(); deleteCount > 0 {
				utils.WarnWithPlatform(u.platform, "失败: 等待上传完成 - 检测到上传出错，准备重试")
				if retryCount >= u.config.MaxUploadRetries {
					return fmt.Errorf("失败: 等待上传完成 - 已达到最大重试次数 %d", u.config.MaxUploadRetries)
				}
				if err := u.handleUploadError(page, videoPath); err != nil {
					return fmt.Errorf("失败: 等待上传完成 - 重试上传失败: %w", err)
				}
				retryCount++
				uploadStartTime = time.Now()
			}
		}

		time.Sleep(u.config.UploadCheckInterval)
	}

	return fmt.Errorf("失败: 等待上传完成 - 上传超时")
}

func (u *Uploader) handleUploadError(page playwright.Page, videoPath string) error {
	deleteBtn := page.Locator("div.media-status-content div.tag-inner:has-text('删除')").First()
	if err := deleteBtn.Click(); err != nil {
		return fmt.Errorf("失败: 处理上传错误 - 点击删除按钮失败: %w", err)
	}

	confirmBtn := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "删除", Exact: playwright.Bool(true)}).First()
	if err := confirmBtn.Click(); err != nil {
		return fmt.Errorf("失败: 处理上传错误 - 点击确认删除失败: %w", err)
	}

	time.Sleep(1 * time.Second)

	fileInput := u.getVideoFileInput(page)
	if err := fileInput.SetInputFiles(videoPath); err != nil {
		return fmt.Errorf("失败: 处理上传错误 - 重新上传视频失败: %w", err)
	}

	return nil
}
