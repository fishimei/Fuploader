package bilibili

import (
	"fmt"
	"time"

	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

func (u *Uploader) setCover(page playwright.Page, thumbnail string) (bool, error) {
	coverMain := page.Locator(`div.cover-main >> span.edit-text:text("封面设置")`).First()
	if err := coverMain.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds()))}); err != nil {
		coverMain = page.Locator(`div.cover-main`).First()
		if err := coverMain.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds()))}); err != nil {
			utils.WarnWithPlatform(u.platform, "失败: 设置封面 - 未找到封面区域")
			return false, nil
		}
	}

	if err := coverMain.ScrollIntoViewIfNeeded(); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置封面 - 滚动到封面区域失败: %v", err))
	}

	if err := coverMain.Click(playwright.LocatorClickOptions{Force: playwright.Bool(true)}); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置封面 - 点击封面区域失败: %v", err))
		return false, nil
	}

	if _, err := page.WaitForSelector(`div.cover-editor`, playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		utils.WarnWithPlatform(u.platform, "失败: 设置封面 - 等待封面编辑器超时")
		return false, nil
	}

	if thumbnail != "" {
		if err := u.uploadCoverImage(page, thumbnail); err != nil {
			utils.WarnWithPlatform(u.platform, err.Error())
			page.Keyboard().Press("Escape")
			return false, nil
		}
	}

	confirmBtn := page.Locator(`div.button.submit:text("完成")`).First()
	if err := confirmBtn.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds()))}); err != nil {
		confirmBtn = page.Locator(`div.cover-editor-button >> div.button.submit`).First()
	}

	if count, _ := confirmBtn.Count(); count > 0 {
		if err := confirmBtn.Click(); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置封面 - 点击完成按钮失败: %v", err))
			page.Keyboard().Press("Escape")
			return false, nil
		}
	} else {
		utils.WarnWithPlatform(u.platform, "失败: 设置封面 - 未找到完成按钮")
		page.Keyboard().Press("Escape")
		return false, nil
	}

	return u.verifyCoverFilled(page)
}

func (u *Uploader) uploadCoverImage(page playwright.Page, thumbnail string) error {
	coverInput := page.Locator(`input[type="file"][accept="image/png, image/jpeg"]`).First()
	if count, _ := coverInput.Count(); count == 0 {
		coverInput = page.Locator(`input[type="file"]`).First()
	}
	if count, _ := coverInput.Count(); count == 0 {
		return fmt.Errorf("失败: 设置封面 - 未找到封面文件输入框")
	}

	if err := coverInput.SetInputFiles(thumbnail); err != nil {
		return fmt.Errorf("失败: 设置封面 - 上传封面失败: %w", err)
	}

	return nil
}

func (u *Uploader) verifyCoverFilled(page playwright.Page) (bool, error) {
	coverCheckStart := time.Now()

	for time.Since(coverCheckStart) < u.config.CoverCheckTimeout {
		coverWithSuccess := page.Locator(`div.cover-main-img.success`).First()
		if count, _ := coverWithSuccess.Count(); count > 0 {
			if isVisible, _ := coverWithSuccess.IsVisible(); isVisible {
				return true, nil
			}
		}

		hasBackground, err := page.Evaluate(`() => {
			const cover = document.querySelector('div.cover-main-img');
			return cover && cover.style.backgroundImage && cover.style.backgroundImage !== '' && cover.style.backgroundImage !== 'none';
		}`)
		if err == nil && hasBackground.(bool) {
			return true, nil
		}

		coverText := page.Locator(`div.cover-main >> span.edit-text:text("封面设置")`).First()
		if count, _ := coverText.Count(); count == 0 {
			return true, nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return false, nil
}
