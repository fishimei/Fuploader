package kuaishou

import (
	"fmt"
	"os"
	"time"

	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

func (u *Uploader) setCover(page playwright.Page, coverPath string) error {
	if _, err := os.Stat(coverPath); err != nil {
		return fmt.Errorf("封面文件不存在: %v", err)
	}

	coverSettingBtn := page.Locator(`div:has-text("封面设置")`).First()
	if err := coverSettingBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("未找到封面设置按钮: %v", err)
	}
	if err := coverSettingBtn.Click(); err != nil {
		return fmt.Errorf("点击封面设置按钮失败: %v", err)
	}
	time.Sleep(1 * time.Second)

	utils.InfoWithPlatform(u.platform, "设置封面...")

	uploadCoverTab := page.Locator(`div:has-text("上传封面")`).First()
	if err := uploadCoverTab.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("未找到上传封面标签: %v", err)
	}
	if err := uploadCoverTab.Click(); err != nil {
		return fmt.Errorf("点击上传封面标签失败: %v", err)
	}
	time.Sleep(1 * time.Second)

	coverInput := page.Locator(`input[type="file"][accept^="image/"]`).First()
	if err := coverInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("未找到封面文件输入框: %v", err)
	}
	if err := coverInput.SetInputFiles(coverPath); err != nil {
		return fmt.Errorf("上传封面失败: %v", err)
	}
	time.Sleep(2 * time.Second)

	confirmBtn := page.Locator(`span:has-text("确认")`).First()
	if err := confirmBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("未找到确认按钮: %v", err)
	}
	if err := confirmBtn.Click(); err != nil {
		return fmt.Errorf("点击确认按钮失败: %v", err)
	}

	utils.InfoWithPlatform(u.platform, "封面设置完成")
	time.Sleep(1 * time.Second)
	return nil
}
