package xiaohongshu

import (
	"fmt"
	"os"
	"time"

	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

type CoverHandler struct {
	config Config
}

func NewCoverHandler(config Config) *CoverHandler {
	return &CoverHandler{config: config}
}

func (h *CoverHandler) SetCover(page playwright.Page, coverPath string) error {
	if coverPath == "" {
		return nil
	}

	if _, err := os.Stat(coverPath); err != nil {
		return fmt.Errorf("失败: 设置封面 - 封面文件不存在: %w", err)
	}

	utils.InfoWithPlatform("xiaohongshu", "设置封面...")

	coverBtn := page.GetByText("选择封面").First()
	if err := coverBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(h.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("失败: 设置封面 - 未找到封面设置按钮: %w", err)
	}

	if err := coverBtn.Click(); err != nil {
		return fmt.Errorf("失败: 设置封面 - 点击封面设置按钮失败: %w", err)
	}
	time.Sleep(2 * time.Second)

	verticalCoverBtn := page.GetByText("设置竖封面").First()
	if count, _ := verticalCoverBtn.Count(); count > 0 {
		verticalCoverBtn.Click()
		time.Sleep(2 * time.Second)
	}

	coverInput := page.Locator("div[class^='semi-upload upload'] >> input.semi-upload-hidden-input").First()
	if err := coverInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(h.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		coverInput = page.Locator("input[type='file']").First()
		if err := coverInput.WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(float64(h.config.ElementWaitTimeout.Milliseconds())),
		}); err != nil {
			return fmt.Errorf("失败: 设置封面 - 未找到封面文件输入框: %w", err)
		}
	}

	if err := coverInput.SetInputFiles(coverPath); err != nil {
		return fmt.Errorf("失败: 设置封面 - 上传封面失败: %w", err)
	}

	time.Sleep(2 * time.Second)

	finishBtn := page.Locator("div[class^='extractFooter'] button:visible:has-text('完成')").First()
	if count, _ := finishBtn.Count(); count > 0 {
		if err := finishBtn.Click(); err != nil {
			utils.WarnWithPlatform("xiaohongshu", fmt.Sprintf("失败: 设置封面 - 点击完成按钮失败: %v", err))
		}
	}
	time.Sleep(2 * time.Second)

	utils.InfoWithPlatform("xiaohongshu", "封面设置完成")
	return nil
}
