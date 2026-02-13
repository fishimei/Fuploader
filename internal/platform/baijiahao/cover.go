package baijiahao

import (
	"fmt"
	"os"
	"time"

	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

func (u *Uploader) setCustomCover(page playwright.Page, coverPath string) error {
	if _, err := os.Stat(coverPath); err != nil {
		return fmt.Errorf("失败: 设置封面 - 文件不存在: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "设置封面...")

	coverArea := page.Locator("div[class^='cover'], div.cover-area, div.cheetah-spin-container").First()
	if err := coverArea.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("失败: 设置封面 - 未找到封面区域: %w", err)
	}

	if err := coverArea.Click(); err != nil {
		return fmt.Errorf("失败: 设置封面 - 点击封面区域失败: %w", err)
	}
	time.Sleep(2 * time.Second)

	coverInput := page.Locator("input[type='file'][accept*='image']").First()
	if err := coverInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("失败: 设置封面 - 未找到文件输入框: %w", err)
	}

	if err := coverInput.SetInputFiles(coverPath); err != nil {
		return fmt.Errorf("失败: 设置封面 - 上传失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "封面上传中...")
	time.Sleep(3 * time.Second)

	confirmBtn := page.Locator("button:has-text('确认'), button:has-text('完成'), button:has-text('确定')").First()
	if count, _ := confirmBtn.Count(); count > 0 {
		if err := confirmBtn.Click(); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置封面 - 点击确认按钮失败: %v", err))
		}
		time.Sleep(1 * time.Second)
	}

	utils.InfoWithPlatform(u.platform, "封面设置完成")
	return nil
}
