package kuaishou

import (
	"context"
	"fmt"
	"time"

	"Fuploader/internal/platform/browser"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

func (u *Uploader) uploadVideo(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext, videoPath string) error {
	utils.InfoWithPlatform(u.platform, "正在上传视频...")

	uploadButton := page.Locator("button[class^='_upload-btn']")
	if err := uploadButton.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("等待上传按钮超时: %v", err)
	}

	fileChooser, err := page.ExpectFileChooser(func() error {
		return uploadButton.Click()
	})
	if err != nil {
		return fmt.Errorf("等待文件选择器失败: %v", err)
	}

	if err := fileChooser.SetFiles(videoPath); err != nil {
		return fmt.Errorf("设置视频文件失败: %v", err)
	}

	utils.InfoWithPlatform(u.platform, "等待视频上传完成...")
	if err := u.waitForUploadComplete(ctx, page, browserCtx); err != nil {
		return fmt.Errorf("等待视频上传失败: %v", err)
	}
	utils.InfoWithPlatform(u.platform, "视频上传完成")

	return nil
}

func (u *Uploader) waitForUploadComplete(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext) error {
	retryInterval := 2 * time.Second

	for retryCount := 0; retryCount < u.config.MaxUploadRetries; retryCount++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("上传已取消")
		default:
		}

		if browserCtx.IsPageClosed() {
			return fmt.Errorf("浏览器已关闭")
		}

		uploadingCount, _ := page.Locator("text=上传中").Count()
		if uploadingCount == 0 {
			successCount, _ := page.Locator("[class*='success'] >> text=上传成功").Count()
			if successCount > 0 {
				return nil
			}

			videoPreview := page.Locator("video, .video-preview, [class*='videoPreview']").First()
			if count, _ := videoPreview.Count(); count > 0 {
				if visible, _ := videoPreview.IsVisible(); visible {
					return nil
				}
			}

			progressCount, _ := page.Locator("[class*='progress'], [class*='uploading']").Count()
			if progressCount == 0 {
				time.Sleep(1 * time.Second)
				finalUploading, _ := page.Locator("text=上传中").Count()
				if finalUploading == 0 {
					return nil
				}
			}
		}

		errorText := page.Locator("text=/上传失败|上传出错|Upload failed/").First()
		if count, _ := errorText.Count(); count > 0 {
			return fmt.Errorf("检测到上传失败")
		}

		time.Sleep(retryInterval)
	}

	return fmt.Errorf("上传超时，已等待%d次检测", u.config.MaxUploadRetries)
}
