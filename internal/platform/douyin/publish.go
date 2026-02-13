package douyin

import (
	"fmt"
	"strings"
	"time"

	"Fuploader/internal/platform/browser"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

func (u *Uploader) setSyncOptions(page playwright.Page, syncToutiao, syncXigua bool) error {
	if syncToutiao {
		toutiaoCheckbox := page.Locator("text=同步到今日头条").Locator("xpath=../input[type='checkbox']").First()
		if count, _ := toutiaoCheckbox.Count(); count > 0 {
			isChecked, _ := toutiaoCheckbox.IsChecked()
			if !isChecked {
				toutiaoCheckbox.Check()
			}
		}
	}

	if syncXigua {
		xiguaCheckbox := page.Locator("text=同步到西瓜视频").Locator("xpath=../input[type='checkbox']").First()
		if count, _ := xiguaCheckbox.Count(); count > 0 {
			isChecked, _ := xiguaCheckbox.IsChecked()
			if !isChecked {
				xiguaCheckbox.Check()
			}
		}
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}

func (u *Uploader) setPermissions(page playwright.Page, allowDownload bool) error {
	if !allowDownload {
		utils.InfoWithPlatform(u.platform, "设置不允许下载...")
		disallowBtn := page.Locator(`span:has-text("不允许")`).First()
		if err := disallowBtn.WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
		}); err != nil {
			return fmt.Errorf("未找到不允许选项: %w", err)
		}
		if err := disallowBtn.Click(); err != nil {
			return fmt.Errorf("点击不允许失败: %w", err)
		}
		utils.InfoWithPlatform(u.platform, "已设置不允许下载")
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}

func (u *Uploader) setScheduleTime(page playwright.Page, scheduleTime string) error {
	scheduleBtn := page.Locator(`span:has-text("定时发布")`).First()
	if err := scheduleBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("未找到定时发布选项: %w", err)
	}
	if err := scheduleBtn.Click(); err != nil {
		return fmt.Errorf("点击定时发布失败: %w", err)
	}
	time.Sleep(1 * time.Second)

	timeInput := page.Locator(`input[format="yyyy-MM-dd HH:mm"]`).First()
	if err := timeInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("未找到时间输入框: %w", err)
	}
	if err := timeInput.Fill(scheduleTime); err != nil {
		return fmt.Errorf("填写定时发布时间失败: %w", err)
	}

	time.Sleep(1 * time.Second)
	return nil
}

func (u *Uploader) publish(page playwright.Page, browserCtx *browser.PooledContext) error {
	for retryCount := 0; retryCount < u.config.MaxPublishRetries; retryCount++ {
		if browserCtx.IsPageClosed() {
			return fmt.Errorf("失败: 发布 - 浏览器已关闭")
		}

		publishBtn := page.GetByRole("button", playwright.PageGetByRoleOptions{
			Name:  "发布",
			Exact: playwright.Bool(true),
		})
		if count, _ := publishBtn.Count(); count > 0 {
			if err := publishBtn.Click(); err != nil {
				utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 发布 - 点击发布按钮失败: %v", err))
			}
		}

		time.Sleep(5 * time.Second)

		url := page.URL()
		if strings.Contains(url, "creator.douyin.com/creator-micro/content/manage") {
			return nil
		}

		coverPrompt := page.GetByText("请设置封面后再发布").First()
		if visible, _ := coverPrompt.IsVisible(); visible {
			recommendCover := page.Locator("[class^='recommendCover-']").First()
			if count, _ := recommendCover.Count(); count > 0 {
				recommendCover.Click()
				time.Sleep(1 * time.Second)
				confirmBtn := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "确定"})
				if count, _ := confirmBtn.Count(); count > 0 {
					confirmBtn.Click()
					time.Sleep(1 * time.Second)
				}
			}
		}

		successText := page.Locator("text=/发布成功|提交成功/").First()
		if count, _ := successText.Count(); count > 0 {
			if visible, _ := successText.IsVisible(); visible {
				return nil
			}
		}
	}

	return fmt.Errorf("失败: 发布 - 发布超时，已重试%d次", u.config.MaxPublishRetries)
}
