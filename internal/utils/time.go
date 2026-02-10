package utils

import (
	"fmt"
	"time"
)

// ScheduleTimeFormats 支持的定时时间格式
var ScheduleTimeFormats = []string{
	"2006-01-02 15:04",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04",
	"2006-01-02T15:04:05",
	"2006/01/02 15:04",
	"2006/01/02 15:04:05",
}

// ParseScheduleTime 解析定时时间（支持多种格式）
func ParseScheduleTime(timeStr string) (time.Time, error) {
	for _, format := range ScheduleTimeFormats {
		if t, err := time.ParseInLocation(format, timeStr, time.Local); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("无法解析时间: %s，支持的格式: %v", timeStr, ScheduleTimeFormats)
}

// FormatScheduleTime 格式化定时时间为标准格式
func FormatScheduleTime(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}

// PlatformTimeFormat 平台时间格式
type PlatformTimeFormat string

const (
	// DouyinFormat 抖音格式
	DouyinFormat PlatformTimeFormat = "2006-01-02 15:04"
	// XiaohongshuFormat 小红书格式
	XiaohongshuFormat PlatformTimeFormat = "2006-01-02 15:04"
	// KuaishouFormat 快手格式
	KuaishouFormat PlatformTimeFormat = "2006-01-02 15:04"
	// TencentFormat 视频号格式
	TencentFormat PlatformTimeFormat = "2006-01-02 15:04"
	// TiktokFormat TikTok格式（5分钟间隔）
	TiktokFormat PlatformTimeFormat = "2006-01-02 15:04"
	// BaijiahaoFormat 百家号格式
	BaijiahaoFormat PlatformTimeFormat = "2006-01-02T15:04"
)

// ToPlatformFormat 转换为平台特定格式
func ToPlatformFormat(t time.Time, platform string) string {
	switch platform {
	case "douyin":
		return t.Format(string(DouyinFormat))
	case "xiaohongshu":
		return t.Format(string(XiaohongshuFormat))
	case "kuaishou":
		return t.Format(string(KuaishouFormat))
	case "tencent":
		return t.Format(string(TencentFormat))
	case "tiktok":
		// TikTok 使用5分钟间隔
		minute := (t.Minute() / 5) * 5
		adjustedTime := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), minute, 0, 0, t.Location())
		return adjustedTime.Format(string(TiktokFormat))
	case "baijiahao":
		return t.Format(string(BaijiahaoFormat))
	default:
		return t.Format("2006-01-02 15:04")
	}
}

// ValidateScheduleTime 验证定时时间是否有效
func ValidateScheduleTime(timeStr string) error {
	t, err := ParseScheduleTime(timeStr)
	if err != nil {
		return err
	}

	// 检查时间是否在未来
	if t.Before(time.Now()) {
		return fmt.Errorf("定时时间必须在未来")
	}

	// 检查时间是否在合理范围内（最多30天）
	if t.After(time.Now().Add(30 * 24 * time.Hour)) {
		return fmt.Errorf("定时时间不能超过30天")
	}

	return nil
}

// GetScheduleDelay 获取距离定时时间的延迟
func GetScheduleDelay(timeStr string) (time.Duration, error) {
	t, err := ParseScheduleTime(timeStr)
	if err != nil {
		return 0, err
	}

	delay := time.Until(t)
	if delay < 0 {
		return 0, fmt.Errorf("定时时间已过期")
	}

	return delay, nil
}

// FormatDuration 格式化持续时间
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d秒", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%d分钟", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%d小时%d分钟", hours, minutes)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%d天%d小时", days, hours)
}

// TruncateToMinute 截断到分钟
func TruncateToMinute(t time.Time) time.Time {
	return t.Truncate(time.Minute)
}

// RoundTo5Minutes 四舍五入到5分钟
func RoundTo5Minutes(t time.Time) time.Time {
	minute := t.Minute()
	roundedMinute := (minute / 5) * 5
	if minute%5 >= 3 {
		roundedMinute += 5
	}
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), roundedMinute, 0, 0, t.Location())
}

// GetNextValidScheduleTime 获取下一个有效的定时时间
func GetNextValidScheduleTime(minDelay time.Duration) time.Time {
	now := time.Now()
	minTime := now.Add(minDelay)

	// 如果最小时间已经过了当前小时的5分钟间隔，则调整到下一个间隔
	minute := minTime.Minute()
	nextMinute := ((minute / 5) + 1) * 5

	if nextMinute >= 60 {
		// 进入下一小时
		return time.Date(minTime.Year(), minTime.Month(), minTime.Day(), minTime.Hour()+1, 0, 0, 0, minTime.Location())
	}

	return time.Date(minTime.Year(), minTime.Month(), minTime.Day(), minTime.Hour(), nextMinute, 0, 0, minTime.Location())
}

// IsValidScheduleTime 检查时间字符串是否是有效的定时时间
func IsValidScheduleTime(timeStr string) bool {
	return ValidateScheduleTime(timeStr) == nil
}
