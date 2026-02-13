package tencent

import (
	"fmt"
	"strings"
	"time"

	"Fuploader/internal/platform/browser"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

func (u *Uploader) fillTitleAndDescription(page playwright.Page, title, description string) error {
	utils.InfoWithPlatform(u.platform, "填写标题...")

	editor := page.Locator("[contenteditable][data-placeholder='添加描述']")
	if err := editor.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("未找到编辑器: %w", err)
	}

	if err := editor.Click(); err != nil {
		return fmt.Errorf("点击编辑器失败: %w", err)
	}
	time.Sleep(500 * time.Millisecond)

	page.Keyboard().Press("Control+KeyA")
	page.Keyboard().Press("Delete")
	page.Keyboard().Type(title)
	page.Keyboard().Press("Enter")

	utils.InfoWithPlatform(u.platform, fmt.Sprintf("标题已填写: %s", title))

	if description != "" {
		utils.InfoWithPlatform(u.platform, "填写描述...")
		page.Keyboard().Press("Enter")
		page.Keyboard().Type(description)
		utils.InfoWithPlatform(u.platform, "描述已填写")
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}

func (u *Uploader) addTags(page playwright.Page, tags []string) error {
	utils.InfoWithPlatform(u.platform, "添加标签...")

	for _, tag := range tags {
		cleanTag := strings.TrimSpace(tag)
		cleanTag = strings.ReplaceAll(cleanTag, "#", "")
		if cleanTag == "" {
			continue
		}

		page.Keyboard().Type("#" + cleanTag)
		page.Keyboard().Press("Space")
		time.Sleep(500 * time.Millisecond)
	}

	utils.InfoWithPlatform(u.platform, "标签添加完成")
	return nil
}

func (u *Uploader) publish(page playwright.Page, browserCtx *browser.PooledContext) error {
	utils.InfoWithPlatform(u.platform, "准备发布...")

	publishBtn := page.Locator("button.weui-desktop-btn_primary:has-text('发表')").First()
	if err := publishBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("失败: 准备发布 - 未找到发表按钮: %w", err)
	}

	if err := publishBtn.Click(); err != nil {
		return fmt.Errorf("失败: 准备发布 - 点击发表按钮失败: %w", err)
	}

	publishTimeout := u.config.SubmitCheckTimeout
	publishStart := time.Now()

	for time.Since(publishStart) < publishTimeout {
		if browserCtx.IsPageClosed() {
			return fmt.Errorf("失败: 发布 - 浏览器已关闭")
		}

		url := page.URL()
		if strings.Contains(url, "post/list") {
			utils.SuccessWithPlatform(u.platform, "发布成功")
			return nil
		}

		successText := page.Locator("text=/发表成功|发布成功/").First()
		if count, _ := successText.Count(); count > 0 {
			if visible, _ := successText.IsVisible(); visible {
				utils.SuccessWithPlatform(u.platform, "发布成功")
				return nil
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("失败: 发布 - 发表超时")
}

func (u *Uploader) saveDraft(page playwright.Page, browserCtx *browser.PooledContext) error {
	draftBtn := page.Locator("div.form-btns button:has-text('保存草稿')").First()
	if err := draftBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("失败: 保存草稿 - 未找到保存草稿按钮: %w", err)
	}

	if count, _ := draftBtn.Count(); count > 0 {
		if err := draftBtn.Click(); err != nil {
			return fmt.Errorf("失败: 保存草稿 - 点击保存草稿按钮失败: %w", err)
		}
	}

	draftTimeout := u.config.SubmitCheckTimeout
	draftStart := time.Now()

	for time.Since(draftStart) < draftTimeout {
		if browserCtx.IsPageClosed() {
			return fmt.Errorf("失败: 保存草稿 - 浏览器已关闭")
		}

		url := page.URL()
		if strings.Contains(url, "post/list") || strings.Contains(url, "draft") {
			utils.SuccessWithPlatform(u.platform, "保存草稿成功")
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("失败: 保存草稿 - 保存草稿超时")
}
