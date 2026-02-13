package baijiahao

import (
	"fmt"
	"time"

	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

func (u *Uploader) setScheduleTime(page playwright.Page, scheduleTime string) error {
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("设置定时发布: %s", scheduleTime))

	targetTime, err := time.Parse("2006-01-02 15:04", scheduleTime)
	if err != nil {
		return fmt.Errorf("失败: 设置定时发布 - 解析时间失败: %w", err)
	}

	page.Locator(`span:has-text("定时发布")`).First().Click()
	time.Sleep(1 * time.Second)

	monthDay := fmt.Sprintf("%d月%d日", targetTime.Month(), targetTime.Day())
	hour := fmt.Sprintf("%d", targetTime.Hour())
	minute := fmt.Sprintf("%d", targetTime.Minute())

	page.Locator(fmt.Sprintf(`div:has-text("选择日期") + div span.cheetah-select-selection-item[title="%s"]`, monthDay)).First().Click()
	time.Sleep(500 * time.Millisecond)

	page.Locator(fmt.Sprintf(`div:has-text("小时") ~ div span:has-text("%s")`, hour)).First().Click()
	time.Sleep(300 * time.Millisecond)
	page.Locator(`div:has-text("小时") ~ div span:has-text("点")`).First().Click()
	time.Sleep(300 * time.Millisecond)

	page.Locator(fmt.Sprintf(`div:has-text("分钟") ~ div span:has-text("%s")`, minute)).First().Click()
	time.Sleep(300 * time.Millisecond)
	page.Locator(`div:has-text("分钟") ~ div span:has-text("分")`).First().Click()
	time.Sleep(300 * time.Millisecond)

	page.Locator(`button:has-text("定时发布")`).First().Click()

	utils.InfoWithPlatform(u.platform, "定时发布设置完成")
	time.Sleep(1 * time.Second)
	return nil
}
