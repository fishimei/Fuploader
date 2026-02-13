package kuaishou

import (
	"fmt"
	"time"

	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

func (u *Uploader) setScheduleTime(page playwright.Page, scheduleTime string) error {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("设置定时发布时间: %s", scheduleTime))

	targetTime, err := time.Parse("2006-01-02 15:04:05", scheduleTime)
	if err != nil {
		targetTime, err = time.Parse("2006-01-02 15:04", scheduleTime)
		if err != nil {
			return fmt.Errorf("解析时间失败: %v", err)
		}
	}

	scheduleLabel := page.Locator("label:has-text('发布时间')")
	if err := scheduleLabel.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("未找到发布时间选项: %v", err)
	}

	scheduleRadio := scheduleLabel.Locator("xpath=following-sibling::div").Locator(".ant-radio-input").Nth(1)
	if err := scheduleRadio.Click(); err != nil {
		scheduleText := page.GetByText("定时发布")
		if err := scheduleText.Click(); err != nil {
			return fmt.Errorf("点击定时发布失败: %v", err)
		}
	}
	time.Sleep(1 * time.Second)

	scheduleInput := page.Locator(`input[placeholder="选择日期和时间"]`).First()
	if err := scheduleInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("未找到定时发布输入框: %v", err)
	}
	if err := scheduleInput.Click(); err != nil {
		return fmt.Errorf("点击定时发布输入框失败: %v", err)
	}
	time.Sleep(1 * time.Second)

	dateStr := targetTime.Format("2006-01-02")
	dateCell := page.Locator(fmt.Sprintf(`td[title="%s"] div.ant-picker-cell-inner`, dateStr)).First()
	if err := dateCell.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("未找到日期单元格 %s: %v", dateStr, err)
	}
	if err := dateCell.Click(); err != nil {
		return fmt.Errorf("点击日期单元格失败: %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	hourStr := targetTime.Format("15")
	hourCell := page.Locator(fmt.Sprintf(`div.ant-picker-time-panel-column >> div.ant-picker-time-panel-cell-inner:has-text("%s")`, hourStr)).First()
	if count, _ := hourCell.Count(); count > 0 {
		if err := hourCell.Click(); err != nil {
			return fmt.Errorf("选择小时失败: %v", err)
		}
		time.Sleep(300 * time.Millisecond)
	}

	minuteStr := targetTime.Format("04")
	minuteCell := page.Locator(fmt.Sprintf(`div.ant-picker-time-panel-column >> div.ant-picker-time-panel-cell-inner:has-text("%s")`, minuteStr)).Nth(1)
	if count, _ := minuteCell.Count(); count > 0 {
		if err := minuteCell.Click(); err != nil {
			return fmt.Errorf("选择分钟失败: %v", err)
		}
		time.Sleep(300 * time.Millisecond)
	}

	secondStr := targetTime.Format("05")
	secondCell := page.Locator(fmt.Sprintf(`div.ant-picker-time-panel-column >> div.ant-picker-time-panel-cell-inner:has-text("%s")`, secondStr)).Nth(2)
	if count, _ := secondCell.Count(); count > 0 {
		if err := secondCell.Click(); err != nil {
			return fmt.Errorf("选择秒失败: %v", err)
		}
		time.Sleep(300 * time.Millisecond)
	}

	confirmBtn := page.Locator(`span:has-text("确定")`).First()
	if err := confirmBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("未找到确定按钮: %v", err)
	}
	if err := confirmBtn.Click(); err != nil {
		return fmt.Errorf("点击确定按钮失败: %v", err)
	}
	time.Sleep(1 * time.Second)
	return nil
}
