package xiaohongshu

import (
	"context"
	"fmt"
	"strings"
	"time"

	"Fuploader/internal/platform/browser"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

func (u *Uploader) uploadVideo(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext, videoPath string) error {
	utils.InfoWithPlatform(u.platform, "正在上传视频...")

	input := page.Locator(`div.drag-over input.upload-input[type="file"][accept*=".mp4"]`).First()
	if err := input.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		input = page.Locator("input[type='file']").First()
		if err := input.WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
		}); err != nil {
			return fmt.Errorf("失败: 上传视频 - 未找到文件输入框: %w", err)
		}
	}

	if err := input.SetInputFiles(videoPath); err != nil {
		return fmt.Errorf("失败: 上传视频 - 设置视频文件失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "等待视频上传完成...")
	if err := u.waitForUploadComplete(ctx, page, browserCtx, videoPath); err != nil {
		return err
	}

	return nil
}

func (u *Uploader) waitForUploadComplete(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext, videoPath string) error {
	uploadStartTime := time.Now()

	for time.Since(uploadStartTime) < u.config.UploadTimeout {
		select {
		case <-ctx.Done():
			return fmt.Errorf("失败: 等待视频上传 - 上传已取消")
		default:
		}

		if browserCtx.IsPageClosed() {
			return fmt.Errorf("失败: 等待视频上传 - 浏览器已关闭")
		}

		uploadSuccess, err := u.detectUploadSuccess(page)
		if err == nil && uploadSuccess {
			utils.InfoWithPlatform(u.platform, "视频上传完成")
			return nil
		}

		reuploadCount, _ := page.Locator("[class^=\"long-card\"] div:has-text(\"重新上传\")").Count()
		if reuploadCount > 0 {
			utils.InfoWithPlatform(u.platform, "视频上传完成")
			return nil
		}

		videoPreview := page.Locator("video, .video-preview, [class*='preview']").First()
		if count, _ := videoPreview.Count(); count > 0 {
			if visible, _ := videoPreview.IsVisible(); visible {
				utils.InfoWithPlatform(u.platform, "视频上传完成")
				return nil
			}
		}

		errorCount, _ := page.Locator("div.progress-div > div:has-text(\"上传失败\")").Count()
		if errorCount > 0 {
			utils.WarnWithPlatform(u.platform, "失败: 上传视频 - 检测到上传失败，尝试重试...")
			retryInput := page.Locator("div.progress-div [class^=\"upload-btn-input\"]")
			if err := retryInput.SetInputFiles(videoPath); err != nil {
				retryInput = page.Locator("input[type='file']")
				if err := retryInput.SetInputFiles(videoPath); err != nil {
					return fmt.Errorf("失败: 上传视频 - 重试上传失败: %w", err)
				}
			}
		}

		time.Sleep(u.config.UploadCheckInterval)
	}

	return fmt.Errorf("失败: 等待视频上传 - 上传超时")
}

func (u *Uploader) detectUploadSuccess(page playwright.Page) (bool, error) {
	uploadInput, err := page.WaitForSelector("input.upload-input", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(3000),
	})
	if err != nil {
		return false, err
	}

	previewNew, err := uploadInput.QuerySelector("xpath=following-sibling::div[contains(@class, 'preview-new')]")
	if err != nil || previewNew == nil {
		return false, fmt.Errorf("失败: 检测上传状态 - 未找到预览区域")
	}

	stageElements, err := previewNew.QuerySelectorAll("div.stage")
	if err != nil || len(stageElements) == 0 {
		return false, fmt.Errorf("失败: 检测上传状态 - 未找到stage元素")
	}

	for _, stage := range stageElements {
		textContent, err := stage.TextContent()
		if err != nil {
			continue
		}
		if strings.Contains(textContent, "上传成功") {
			return true, nil
		}
	}

	return false, fmt.Errorf("失败: 检测上传状态 - 未检测到上传成功")
}
