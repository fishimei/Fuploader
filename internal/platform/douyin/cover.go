package douyin

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
	utils.InfoWithPlatform("douyin", "设置封面...")

	coverBtn := page.GetByText("选择封面").First()
	if err := coverBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(h.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("未找到封面设置按钮: %w", err)
	}

	if err := coverBtn.Click(); err != nil {
		return fmt.Errorf("点击封面设置按钮失败: %w", err)
	}
	time.Sleep(2 * time.Second)

	if coverPath != "" {
		if _, err := os.Stat(coverPath); err == nil {
			utils.InfoWithPlatform("douyin", "上传自定义封面...")

			coverInput := page.Locator(`input[type="file"][accept^="image/"].semi-upload-hidden-input`).First()
			if err := coverInput.WaitFor(playwright.LocatorWaitForOptions{
				Timeout: playwright.Float(float64(h.config.ElementWaitTimeout.Milliseconds())),
			}); err != nil {
				utils.WarnWithPlatform("douyin", fmt.Sprintf("未找到封面上传输入框: %v", err))
			} else {
				if err := coverInput.SetInputFiles(coverPath); err != nil {
					utils.WarnWithPlatform("douyin", fmt.Sprintf("上传封面失败: %v", err))
				} else {
					utils.InfoWithPlatform("douyin", "封面上传中...")
					time.Sleep(3 * time.Second)
				}
			}
		}
	}

	verticalBtn := page.Locator(`button:has-text("设置竖封面")`).First()
	if err := verticalBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(h.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		utils.WarnWithPlatform("douyin", "未找到设置竖封面按钮")
	} else {
		if err := verticalBtn.Click(); err != nil {
			utils.WarnWithPlatform("douyin", fmt.Sprintf("点击设置竖封面按钮失败: %v", err))
		} else {
			utils.InfoWithPlatform("douyin", "已切换到竖封面")
		}
		time.Sleep(2 * time.Second)
	}

	finishBtn := page.Locator(`span:has-text("完成")`).First()
	if err := finishBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(h.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		utils.WarnWithPlatform("douyin", fmt.Sprintf("未找到完成按钮: %v", err))
	} else {
		if err := finishBtn.Click(); err != nil {
			utils.WarnWithPlatform("douyin", fmt.Sprintf("点击完成按钮失败: %v", err))
		} else {
			utils.InfoWithPlatform("douyin", "已点击完成按钮")
		}
	}
	time.Sleep(2 * time.Second)

	utils.InfoWithPlatform("douyin", "封面设置完成")
	return nil
}
