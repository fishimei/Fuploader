package tencent

import (
	"fmt"
	"strings"
	"time"

	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

func (u *Uploader) setScheduleTime(page playwright.Page, scheduleTime string) error {
	targetTime, err := utils.ParseScheduleTime(scheduleTime)
	if err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 解析时间失败: %w", err)
	}

	scheduleLabel := page.Locator("label").Filter(playwright.LocatorFilterOptions{HasText: playwright.String("定时")}).Nth(1)
	if err := scheduleLabel.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 未找到定时发表选项: %w", err)
	}

	if err := scheduleLabel.Click(); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 点击定时发表失败: %w", err)
	}
	time.Sleep(1 * time.Second)

	timeInput := page.Locator("input.weui-desktop-form-input__input[placeholder='请选择发表时间']").First()
	if err := timeInput.Click(); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 点击时间输入框失败: %w", err)
	}
	time.Sleep(1 * time.Second)

	strMonth := fmt.Sprintf("%02d月", targetTime.Month())

	pageMonth, err := page.InnerText("span.weui-desktop-picker__panel__label:has-text('月')")
	if err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 获取当前月份失败: %w", err)
	}

	for pageMonth != strMonth {
		nextMonthBtn := page.Locator("button.weui-desktop-btn__icon__right").First()
		if err := nextMonthBtn.Click(); err != nil {
			return fmt.Errorf("失败: 设置定时发布 - 点击下个月按钮失败: %w", err)
		}
		time.Sleep(500 * time.Millisecond)

		pageMonth, err = page.InnerText("span.weui-desktop-picker__panel__label:has-text('月')")
		if err != nil {
			return fmt.Errorf("失败: 设置定时发布 - 获取当前月份失败: %w", err)
		}
	}

	elements, err := page.QuerySelectorAll("table.weui-desktop-picker__table a")
	if err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 获取日期元素失败: %w", err)
	}

	for _, element := range elements {
		className, err := element.Evaluate("el => el.className")
		if err != nil {
			continue
		}

		classNameStr, ok := className.(string)
		if ok && strings.Contains(classNameStr, "weui-desktop-picker__disabled") {
			continue
		}

		text, err := element.InnerText()
		if err != nil {
			continue
		}

		if strings.TrimSpace(text) == fmt.Sprintf("%d", targetTime.Day()) {
			if err := element.Click(); err != nil {
				return fmt.Errorf("失败: 设置定时发布 - 点击日期失败: %w", err)
			}
			break
		}
	}

	hourInput := page.Locator("input.weui-desktop-form-input__input[placeholder='请选择时间']").First()
	if err := hourInput.Click(); err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 点击时间选择框失败: %w", err)
	}
	time.Sleep(500 * time.Millisecond)

	page.Keyboard().Press("Control+KeyA")
	page.Keyboard().Type(fmt.Sprintf("%d", targetTime.Hour()))

	page.Locator("[contenteditable][data-placeholder='添加描述']").Click()

	time.Sleep(1 * time.Second)
	return nil
}
