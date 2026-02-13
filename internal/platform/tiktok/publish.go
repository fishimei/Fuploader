package tiktok

import (
	"context"
	"fmt"
	"strings"
	"time"

	"Fuploader/internal/platform/browser"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

func (u *Uploader) publish(ctx context.Context, page playwright.Page, locatorBase playwright.Locator, browserCtx *browser.PooledContext) error {
	locators := GetLocators()
	publishTimeout := 2 * time.Minute
	publishStartTime := time.Now()

	for time.Since(publishStartTime) < publishTimeout {
		select {
		case <-ctx.Done():
			return fmt.Errorf("发布已取消")
		default:
		}

		if browserCtx.IsPageClosed() {
			return fmt.Errorf("浏览器已关闭")
		}

		publishBtn := locatorBase.Locator("div.btn-post")
		if count, _ := publishBtn.Count(); count > 0 {
			publishBtn.Click()
		}

		time.Sleep(3 * time.Second)

		successLocator := locatorBase.Locator(locators.SuccessModal)
		if visible, _ := successLocator.IsVisible(); visible {
			return nil
		}

		if count, _ := successLocator.Count(); count > 0 {
			return nil
		}

		url := page.URL()
		if strings.Contains(url, "tiktokstudio/content") {
			return nil
		}

		successText := locatorBase.Locator(`text=/发布成功|提交成功|Published|Success/`).First()
		if count, _ := successText.Count(); count > 0 {
			if visible, _ := successText.IsVisible(); visible {
				return nil
			}
		}

		utils.InfoWithPlatform(u.platform, "等待发布完成...")

		select {
		case <-ctx.Done():
			return fmt.Errorf("发布已取消")
		case <-time.After(500 * time.Millisecond):
		}
	}

	return fmt.Errorf("发布超时")
}
