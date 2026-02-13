package tiktok

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

func (u *Uploader) setScheduleTime(locatorBase playwright.Locator, scheduleTime string) error {
	utils.InfoWithPlatform(u.platform, "设置定时发布...")

	publishDate, err := time.Parse("2006-01-02 15:04", scheduleTime)
	if err != nil {
		publishDate, err = utils.ParseScheduleTime(scheduleTime)
		if err != nil {
			return fmt.Errorf("解析时间失败: %w", err)
		}
	}

	locators := GetLocators()

	scheduleBtn := locatorBase.GetByLabel("Schedule")
	if err := scheduleBtn.WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateVisible,
	}); err != nil {
		return fmt.Errorf("未找到Schedule按钮: %w", err)
	}

	scheduleBtn.Click()
	time.Sleep(1 * time.Second)

	allowBtn := u.findFirstVisibleLocator(locatorBase, locators.AllowButton)
	if allowBtn != nil {
		if err := allowBtn.Click(); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("点击Allow按钮失败: %v", err))
		}
	}

	scheduledPicker := locatorBase.Locator(locators.DatePicker)
	calendarBtn := scheduledPicker.Nth(1)
	calendarBtn.Click()
	time.Sleep(500 * time.Millisecond)

	calendarWrapper := locatorBase.Locator(locators.CalendarWrapper)
	monthTitle := calendarWrapper.Locator(locators.MonthTitle)
	monthText, _ := monthTitle.TextContent()
	currentMonth := parseMonth(monthText)
	targetMonth := int(publishDate.Month())

	if currentMonth != targetMonth {
		if err := u.switchMonth(calendarWrapper, currentMonth, targetMonth); err != nil {
			return fmt.Errorf("切换月份失败: %w", err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	validDays := calendarWrapper.Locator(locators.ValidDay)
	count, _ := validDays.Count()
	targetDay := strconv.Itoa(publishDate.Day())
	for i := 0; i < count; i++ {
		dayText, _ := validDays.Nth(i).TextContent()
		if strings.TrimSpace(dayText) == targetDay {
			validDays.Nth(i).Click()
			break
		}
	}

	timePicker := locatorBase.Locator(locators.TimePicker)
	timePicker.Nth(0).Click()
	time.Sleep(500 * time.Millisecond)

	hourStr := publishDate.Format("15")
	hourSelector := fmt.Sprintf("%s:has-text('%s')", locators.HourPicker, hourStr)
	hourElement := locatorBase.Locator(hourSelector)
	hourElement.Click()
	time.Sleep(500 * time.Millisecond)

	timePicker.Nth(0).Click()
	time.Sleep(500 * time.Millisecond)

	correctMinute := int(publishDate.Minute()/5) * 5
	minuteStr := fmt.Sprintf("%02d", correctMinute)
	minuteSelector := fmt.Sprintf("%s:has-text('%s')", locators.MinutePicker, minuteStr)
	minuteElement := locatorBase.Locator(minuteSelector)
	minuteElement.Click()

	uploadTitle := locatorBase.Locator("h1:has-text('Upload video')")
	uploadTitle.Click()

	utils.InfoWithPlatform(u.platform, "定时发布设置完成")
	return nil
}

func (u *Uploader) switchMonth(calendarWrapper playwright.Locator, currentMonth, targetMonth int) error {
	arrows := calendarWrapper.Locator("span.arrow")
	arrowCount, _ := arrows.Count()

	if arrowCount == 0 {
		return fmt.Errorf("未找到月份切换箭头")
	}

	var clickCount int
	if currentMonth < targetMonth {
		clickCount = targetMonth - currentMonth
		if clickCount > 6 {
			clickCount = 12 - clickCount
		}
		for i := 0; i < clickCount && i < int(arrowCount); i++ {
			arrows.Nth(int(arrowCount) - 1).Click()
			time.Sleep(300 * time.Millisecond)
		}
	} else {
		clickCount = currentMonth - targetMonth
		if clickCount > 6 {
			clickCount = 12 - clickCount
		}
		for i := 0; i < clickCount; i++ {
			arrows.Nth(0).Click()
			time.Sleep(300 * time.Millisecond)
		}
	}

	return nil
}

func parseMonth(monthName string) int {
	months := map[string]int{
		"January":   1, "February": 2, "March": 3, "April": 4,
		"May":       5, "June": 6, "July": 7, "August": 8,
		"September": 9, "October": 10, "November": 11, "December": 12,
		"Jan": 1, "Feb": 2, "Mar": 3, "Apr": 4,
		"Jun": 6, "Jul": 7, "Aug": 8, "Sep": 9,
		"Oct": 10, "Nov": 11, "Dec": 12,
		"1月": 1, "2月": 2, "3月": 3, "4月": 4,
		"5月": 5, "6月": 6, "7月": 7, "8月": 8,
		"9月": 9, "10月": 10, "11月": 11, "12月": 12,
	}

	monthName = strings.TrimSpace(monthName)

	if month, ok := months[monthName]; ok {
		return month
	}

	for name, month := range months {
		if strings.Contains(monthName, name) {
			return month
		}
	}

	return 0
}
