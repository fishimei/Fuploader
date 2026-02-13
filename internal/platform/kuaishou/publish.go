package kuaishou

import (
	"fmt"
	"strings"
	"time"

	"Fuploader/internal/platform/browser"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

func (u *Uploader) handleNewFeatureGuide(page playwright.Page) error {
	newFeatureBtn := page.Locator("button[type='button'] span:has-text('我知道了')")
	count, _ := newFeatureBtn.Count()
	if count > 0 {
		if err := newFeatureBtn.Click(); err == nil {
			time.Sleep(1 * time.Second)
		}
	}
	return nil
}

func (u *Uploader) handleSkipPopup(page playwright.Page) error {
	skipBtn := page.Locator(`div[aria-label="Skip"][title="Skip"]`).First()
	count, _ := skipBtn.Count()
	if count > 0 {
		if err := skipBtn.Click(); err != nil {
			return fmt.Errorf("点击Skip按钮失败: %v", err)
		}
		time.Sleep(500 * time.Millisecond)
	}
	return nil
}

func (u *Uploader) setDownloadPermission(page playwright.Page, allowDownload bool) error {
	checkbox := page.Locator(`span:has-text("允许下载此作品")`).First()
	count, _ := checkbox.Count()
	if count == 0 {
		return fmt.Errorf("未找到下载权限设置选项")
	}

	parentCheckbox := checkbox.Locator("xpath=ancestor::label//input[@type='checkbox']").First()
	if parentCount, _ := parentCheckbox.Count(); parentCount > 0 {
		isChecked, _ := parentCheckbox.IsChecked()
		if isChecked != allowDownload {
			if err := checkbox.Click(); err != nil {
				return fmt.Errorf("点击下载权限选项失败: %v", err)
			}
		}
	} else {
		if !allowDownload {
			if err := checkbox.Click(); err != nil {
				return fmt.Errorf("点击下载权限选项失败: %v", err)
			}
		}
	}

	utils.InfoWithPlatform(u.platform, fmt.Sprintf("下载权限已设置: %v", allowDownload))
	time.Sleep(500 * time.Millisecond)
	return nil
}

func (u *Uploader) fillDescription(page playwright.Page, title, description string) error {
	utils.InfoWithPlatform(u.platform, "填写标题...")

	descArea := page.GetByText("描述").Locator("xpath=following-sibling::div")
	if err := descArea.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		descArea = page.Locator("textarea[placeholder*='描述'], div[contenteditable='true']").First()
		if err := descArea.WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
		}); err != nil {
			return fmt.Errorf("未找到描述输入区域: %v", err)
		}
	}

	if err := descArea.Click(); err != nil {
		return fmt.Errorf("点击描述区域失败: %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	page.Keyboard().Press("Control+A")
	page.Keyboard().Press("Delete")
	time.Sleep(300 * time.Millisecond)

	content := ""
	if title != "" {
		content = title
	}
	if description != "" {
		if content != "" {
			content += "\n"
		}
		content += description
	}

	if content != "" {
		page.Keyboard().Type(content)
		utils.InfoWithPlatform(u.platform, fmt.Sprintf("标题已填写: %s", title))
		utils.InfoWithPlatform(u.platform, "描述已填写")
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}

func (u *Uploader) addTags(page playwright.Page, tags []string) error {
	maxTags := 3
	if len(tags) < maxTags {
		maxTags = len(tags)
	}

	utils.InfoWithPlatform(u.platform, "添加标签...")

	for _, tag := range tags[:maxTags] {
		cleanTag := strings.TrimSpace(tag)
		cleanTag = strings.ReplaceAll(cleanTag, "#", "")

		if cleanTag == "" {
			continue
		}

		page.Keyboard().Type(fmt.Sprintf("#%s", cleanTag))
		time.Sleep(1 * time.Second)

		suggestion := page.Locator(`[class*='tag-suggestion'], [class*='topic-item'], [class*='mention-item']`).First()
		if count, _ := suggestion.Count(); count > 0 {
			if err := suggestion.Click(); err == nil {
				utils.InfoWithPlatform(u.platform, fmt.Sprintf("标签添加成功: #%s", cleanTag))
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	utils.InfoWithPlatform(u.platform, "标签添加完成")
	return nil
}

func (u *Uploader) publish(page playwright.Page, browserCtx *browser.PooledContext) error {
	for attempt := 0; attempt < u.config.MaxClickAttempts; attempt++ {
		if browserCtx.IsPageClosed() {
			return fmt.Errorf("浏览器已关闭")
		}

		publishButton := page.GetByText("发布", playwright.PageGetByTextOptions{Exact: playwright.Bool(true)})
		count, _ := publishButton.Count()
		if count > 0 {
			if err := publishButton.Click(); err != nil {
				utils.WarnWithPlatform(u.platform, fmt.Sprintf("点击发布按钮失败: %v", err))
			}
		}

		time.Sleep(1 * time.Second)

		confirmButton := page.GetByText("确认发布")
		confirmCount, _ := confirmButton.Count()
		if confirmCount > 0 {
			if err := confirmButton.Click(); err != nil {
				utils.WarnWithPlatform(u.platform, fmt.Sprintf("点击确认发布失败: %v", err))
			}
		}

		currentURL := page.URL()
		if currentURL == "https://cp.kuaishou.com/article/manage/video?status=2&from=publish" {
			return nil
		}

		successCount, _ := page.Locator("text=发布成功").Count()
		if successCount > 0 {
			if visible, _ := page.Locator("text=发布成功").IsVisible(); visible {
				return nil
			}
		}

		errorText := page.Locator("text=/发布失败|提交失败|错误/").First()
		if count, _ := errorText.Count(); count > 0 {
			if visible, _ := errorText.IsVisible(); visible {
				text, _ := errorText.TextContent()
				return fmt.Errorf("发布失败: %s", text)
			}
		}

		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("发布超时，已尝试%d次", u.config.MaxClickAttempts)
}
