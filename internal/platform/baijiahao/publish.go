package baijiahao

import (
	"fmt"
	"strings"
	"time"

	"Fuploader/internal/platform/browser"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

func (u *Uploader) publish(page playwright.Page, browserCtx *browser.PooledContext) error {
	publishBtn := page.Locator("button:has-text('发布'), button:has-text('立即发布')").First()
	if err := publishBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("失败: 发布 - 未找到发布按钮: %w", err)
	}

	if err := publishBtn.ScrollIntoViewIfNeeded(); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 发布 - 滚动到按钮失败: %v", err))
	}

	urlBeforePublish := page.URL()

	for attempt := 0; attempt < u.config.MaxPublishRetries; attempt++ {
		if browserCtx.IsPageClosed() {
			return fmt.Errorf("失败: 发布 - 浏览器已关闭")
		}

		utils.InfoWithPlatform(u.platform, fmt.Sprintf("第%d次尝试发布...", attempt+1))

		if err := publishBtn.Click(playwright.LocatorClickOptions{
			Force: playwright.Bool(true),
		}); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 发布 - 点击按钮失败: %v", err))
			time.Sleep(2 * time.Second)
			continue
		}

		utils.InfoWithPlatform(u.platform, "已点击发布按钮")
		time.Sleep(3 * time.Second)

		confirmDialog := page.Locator("button:has-text('确定'), button:has-text('确认')").First()
		if count, _ := confirmDialog.Count(); count > 0 {
			if visible, _ := confirmDialog.IsVisible(); visible {
				utils.InfoWithPlatform(u.platform, "处理确认弹窗...")
				confirmDialog.Click()
				time.Sleep(2 * time.Second)
			}
		}

		if err := u.checkPublishResult(page, browserCtx, urlBeforePublish); err == nil {
			return nil
		} else {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 发布 - 检测未通过: %v", err))
		}

		if attempt < u.config.MaxPublishRetries-1 {
			time.Sleep(3 * time.Second)
		}
	}

	return fmt.Errorf("失败: 发布 - 已重试%d次", u.config.MaxPublishRetries)
}

func (u *Uploader) checkPublishResult(page playwright.Page, browserCtx *browser.PooledContext, urlBefore string) error {
	checkStart := time.Now()

	for time.Since(checkStart) < u.config.SubmitCheckTimeout {
		if browserCtx.IsPageClosed() {
			return fmt.Errorf("失败: 检测发布结果 - 浏览器已关闭")
		}

		currentURL := page.URL()

		if strings.Contains(currentURL, "baijiahao.baidu.com/builder/rc/clue") ||
			strings.Contains(currentURL, "baijiahao.baidu.com/builder/rc/manage") {
			return nil
		}

		if currentURL != urlBefore && !strings.Contains(currentURL, "edit") {
			return nil
		}

		successIndicators := []string{"发布成功", "提交成功", "审核中", "稿件已提交"}
		for _, indicator := range successIndicators {
			successText := page.Locator(fmt.Sprintf("text=%s", indicator)).First()
			if count, _ := successText.Count(); count > 0 {
				if visible, _ := successText.IsVisible(); visible {
					return nil
				}
			}
		}

		errorIndicators := []string{"发布失败", "提交失败", "错误", "请完善"}
		for _, indicator := range errorIndicators {
			errorText := page.Locator(fmt.Sprintf("text=%s", indicator)).First()
			if count, _ := errorText.Count(); count > 0 {
				if visible, _ := errorText.IsVisible(); visible {
					text, _ := errorText.TextContent()
					return fmt.Errorf("失败: 检测发布结果 - 页面错误: %s", text)
				}
			}
		}

		time.Sleep(u.config.UploadCheckInterval)
	}

	return fmt.Errorf("失败: 检测发布结果 - 超时")
}
