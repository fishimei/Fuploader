package baijiahao

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

	inputLocator := page.Locator("div[class^='video-main-container'] input[type='file']").First()
	if count, _ := inputLocator.Count(); count == 0 {
		inputLocator = page.Locator("input[type='file']").First()
	}

	if err := inputLocator.SetInputFiles(videoPath); err != nil {
		return fmt.Errorf("失败: 上传视频 - %w", err)
	}

	return u.waitForUploadComplete(ctx, page, browserCtx)
}

func (u *Uploader) waitForUploadComplete(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext) error {
	utils.InfoWithPlatform(u.platform, "等待视频上传完成...")

	uploadStartTime := time.Now()

	for time.Since(uploadStartTime) < u.config.UploadTimeout {
		select {
		case <-ctx.Done():
			return fmt.Errorf("上传已取消")
		default:
		}

		if browserCtx.IsPageClosed() {
			return fmt.Errorf("浏览器已关闭")
		}

		coverImg := page.Locator(`div[class*="cover-container"] img[class*="coverImg"]`).First()
		if count, _ := coverImg.Count(); count > 0 {
			if visible, _ := coverImg.IsVisible(); visible {
				utils.InfoWithPlatform(u.platform, "视频上传完成")
				return nil
			}
		}

		errorText := page.Locator("text=/上传失败|上传出错/").First()
		if count, _ := errorText.Count(); count > 0 {
			if visible, _ := errorText.IsVisible(); visible {
				return fmt.Errorf("失败: 上传视频 - 检测到上传失败")
			}
		}

		time.Sleep(u.config.UploadCheckInterval)
	}

	return fmt.Errorf("失败: 上传视频 - 上传超时")
}
