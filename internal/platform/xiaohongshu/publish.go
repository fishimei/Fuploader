package xiaohongshu

import (
	"fmt"
	"time"

	"Fuploader/internal/platform/browser"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

func (u *Uploader) setScheduleTime(page playwright.Page, scheduleTime string) error {
	utils.InfoWithPlatform(u.platform, "设置定时发布...")

	targetTime, err := time.Parse("2006-01-02 15:04", scheduleTime)
	if err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 解析时间失败: %w", err)
	}

	labelElement := page.Locator("label:has-text('定时发布')")
	if err := labelElement.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 未找到定时发布选项: %w", err)
	}

	if err := labelElement.Click(); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 点击定时发布失败: %w", err)
	}
	time.Sleep(1 * time.Second)

	scheduleInput := page.Locator(".el-input__inner[placeholder=\"选择日期和时间\"]")
	if err := scheduleInput.Click(); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 点击时间输入框失败: %w", err)
	}
	time.Sleep(500 * time.Millisecond)

	timeStr := targetTime.Format("2006-01-02 15:04")
	page.Keyboard().Press("Control+KeyA")
	page.Keyboard().Type(timeStr)
	page.Keyboard().Press("Enter")

	time.Sleep(1 * time.Second)
	return nil
}

func (u *Uploader) publish(page playwright.Page, browserCtx *browser.PooledContext, isScheduled bool) error {
	publishStart := time.Now()

	for retryCount := 0; retryCount < u.config.MaxPublishRetries; retryCount++ {
		if time.Since(publishStart) > u.config.SubmitCheckTimeout {
			return fmt.Errorf("失败: 发布 - 发布超时")
		}

		if browserCtx.IsPageClosed() {
			return fmt.Errorf("失败: 发布 - 浏览器已关闭")
		}

		if isScheduled {
			button := page.Locator("button:has-text('定时发布')")
			if count, _ := button.Count(); count > 0 {
				if err := button.Click(); err != nil {
					utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 发布 - 点击定时发布按钮失败: %v", err))
				}
			}
		} else {
			button := page.Locator("button:has-text('发布')")
			if count, _ := button.Count(); count > 0 {
				if err := button.Click(); err != nil {
					utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 发布 - 点击发布按钮失败: %v", err))
				}
			}
		}

		waitErr := page.WaitForURL("https://creator.xiaohongshu.com/publish/success?**", playwright.PageWaitForURLOptions{
			Timeout: playwright.Float(3000),
		})
		if waitErr == nil {
			return nil
		}

		_, _ = page.Screenshot(playwright.PageScreenshotOptions{
			FullPage: playwright.Bool(true),
		})

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("失败: 发布 - 发布超时，已重试%d次", u.config.MaxPublishRetries)
}
