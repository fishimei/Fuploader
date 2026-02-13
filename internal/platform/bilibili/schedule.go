package bilibili

import (
	"fmt"
	"time"

	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

func (u *Uploader) setScheduleTime(page playwright.Page, scheduleTime string) error {
	var targetTime time.Time
	var err error

	targetTime, err = time.Parse("2006-01-02 15:04", scheduleTime)
	if err != nil {
		targetTime, err = time.Parse("2006-01-02T15:04:05", scheduleTime)
		if err != nil {
			targetTime, err = time.Parse(time.RFC3339, scheduleTime)
			if err != nil {
				return fmt.Errorf("失败: 设置定时发布 - 解析时间失败，支持格式: 2006-01-02 15:04 或 2006-01-02T15:04:05 或 RFC3339: %w", err)
			}
		}
	}

	now := time.Now()
	minTime := now.Add(2 * time.Hour)
	maxTime := now.Add(15 * 24 * time.Hour)

	if targetTime.Before(minTime) {
		return fmt.Errorf("失败: 设置定时发布 - 定时时间必须至少提前2小时")
	}
	if targetTime.After(maxTime) {
		return fmt.Errorf("失败: 设置定时发布 - 定时时间不能超过15天")
	}

	switchContainer := page.Locator(`div.switch-container.switch-container-active`).First()
	if err := switchContainer.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds()))}); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 未找到定时开关: %w", err)
	}

	if err := switchContainer.Click(); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 点击定时开关失败: %w", err)
	}
	utils.InfoWithPlatform(u.platform, "已开启定时发布")

	if _, err := page.WaitForSelector(`div.date-picker-date-wrp`, playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 等待日期选择器超时: %w", err)
	}

	datePicker := page.Locator(`div.date-picker-date-wrp`).First()
	if err := datePicker.Click(); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 点击日期选择器失败: %w", err)
	}

	dateStr := targetTime.Format("2006-01-02")
	dateCell := page.Locator(fmt.Sprintf(`div[aria-label="%s"]`, dateStr)).First()
	if err := dateCell.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds()))}); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 未找到目标日期: %w", err)
	}

	if err := dateCell.Click(); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 选择日期失败: %w", err)
	}

	timePicker := page.Locator(`div.date-picker-timer`).First()
	if err := timePicker.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds()))}); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 未找到时间选择器: %w", err)
	}

	if err := timePicker.Click(); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 点击时间选择器失败: %w", err)
	}

	timeStr := targetTime.Format("15:04")
	timeCell := page.Locator(fmt.Sprintf(`div[aria-label="%s"]`, timeStr)).First()
	if err := timeCell.WaitFor(playwright.LocatorWaitForOptions{Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds()))}); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 未找到目标时间: %w", err)
	}

	if err := timeCell.Click(); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 选择时间失败: %w", err)
	}

	return nil
}
