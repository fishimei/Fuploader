package douyin

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"Fuploader/internal/platform/browser"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

func (u *Uploader) uploadVideo(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext, videoPath string) error {
	inputLocator := page.Locator("div[class^='container'] input[type='file']").First()
	if err := inputLocator.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		inputLocator = page.Locator("input[type='file']").First()
		if err := inputLocator.WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
		}); err != nil {
			return fmt.Errorf("失败: 上传视频 - 未找到文件输入框: %w", err)
		}
	}

	if err := inputLocator.SetInputFiles(videoPath); err != nil {
		return fmt.Errorf("失败: 上传视频 - 设置视频文件失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "等待视频上传完成...")
	if err := u.waitForUploadComplete(ctx, page, browserCtx); err != nil {
		return err
	}

	return nil
}

func (u *Uploader) waitForUploadComplete(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext) error {
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

		videoPreview := page.Locator("video, .video-preview, [class*='videoPreview'], div[class*='player']").First()
		if count, _ := videoPreview.Count(); count > 0 {
			if visible, _ := videoPreview.IsVisible(); visible {
				utils.InfoWithPlatform(u.platform, "视频上传完成")
				return nil
			}
		}

		progressBar := page.Locator("div[class*='progress'], div[class*='uploading']").First()
		if count, _ := progressBar.Count(); count == 0 {
			videoInfo := page.Locator("div[class*='video-info'], div[class*='mediaInfo']").First()
			if count, _ := videoInfo.Count(); count > 0 {
				utils.InfoWithPlatform(u.platform, "视频上传完成")
				return nil
			}
		}

		successText := page.Locator("text=/上传成功|上传完成/").First()
		if count, _ := successText.Count(); count > 0 {
			if visible, _ := successText.IsVisible(); visible {
				utils.InfoWithPlatform(u.platform, "视频上传完成")
				return nil
			}
		}

		errorText := page.Locator("text=/上传失败|上传出错/").First()
		if count, _ := errorText.Count(); count > 0 {
			return fmt.Errorf("失败: 上传视频 - 检测到上传失败")
		}

		time.Sleep(u.config.UploadCheckInterval)
	}

	return fmt.Errorf("失败: 上传视频 - 上传超时")
}

func (u *Uploader) fillTitle(page playwright.Page, title string) error {
	if title == "" {
		return nil
	}

	utils.InfoWithPlatform(u.platform, "填写标题...")

	truncatedTitle := TruncateString(title, u.config.TitleMaxLength)

	titleInput := page.Locator(`input[placeholder="填写作品标题，为作品获得更多流量"]`).First()
	if err := titleInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("未找到标题输入框: %w", err)
	}

	if err := titleInput.Fill(truncatedTitle); err != nil {
		return fmt.Errorf("填写标题失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, fmt.Sprintf("标题已填写: %s", truncatedTitle))
	time.Sleep(500 * time.Millisecond)
	return nil
}

func (u *Uploader) fillDescription(page playwright.Page, description string) error {
	utils.InfoWithPlatform(u.platform, "填写描述...")

	descContainer := page.Locator(".zone-container").First()
	if err := descContainer.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("未找到描述输入框: %w", err)
	}

	if err := descContainer.Fill(description); err != nil {
		return fmt.Errorf("填写描述失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "描述已填写")
	time.Sleep(500 * time.Millisecond)
	return nil
}

func (u *Uploader) addTags(page playwright.Page, tags []string) error {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("添加%d个标签...", len(tags)))

	tagContainer := page.Locator(".zone-container").First()
	if err := tagContainer.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		tagContainer = page.Locator("div[class*='tag'], div[class*='topic']").First()
		if err := tagContainer.WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
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

func (u *Uploader) addProductLink(page playwright.Page, productLink, productTitle string) error {
	utils.InfoWithPlatform(u.platform, "添加商品链接...")

	addTagBtn := page.GetByText("添加标签").First()
	if err := addTagBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("未找到添加标签按钮: %w", err)
	}

	if err := addTagBtn.Click(); err != nil {
		return fmt.Errorf("点击添加标签失败: %w", err)
	}
	time.Sleep(1 * time.Second)

	cartBtn := page.GetByText("购物车").First()
	if count, _ := cartBtn.Count(); count > 0 {
		cartBtn.Click()
		time.Sleep(1 * time.Second)
	}

	linkInput := page.Locator("input[placeholder='添加商品链接']").First()
	if err := linkInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("未找到商品链接输入框: %w", err)
	}

	if err := linkInput.Fill(productLink); err != nil {
		return fmt.Errorf("填写商品链接失败: %w", err)
	}

	if productTitle != "" {
		shortTitle := TruncateString(productTitle, u.config.ShortTitleMaxLength)

		titleInput := page.Locator("input[placeholder*='短标题']").First()
		if count, _ := titleInput.Count(); count > 0 {
			titleInput.Fill(shortTitle)
		}
	}

	utils.InfoWithPlatform(u.platform, "商品链接已添加")
	time.Sleep(1 * time.Second)
	return nil
}

func (u *Uploader) checkVideoExists(videoPath string) error {
	if _, err := os.Stat(videoPath); err != nil {
		return fmt.Errorf("视频文件不存在: %w", err)
	}
	return nil
}
