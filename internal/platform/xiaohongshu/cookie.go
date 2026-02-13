package xiaohongshu

import (
	"Fuploader/internal/platform/browser"
)

func GetCookieConfig() (browser.PlatformCookieConfig, bool) {
	return browser.GetCookieConfig("xiaohongshu")
}
