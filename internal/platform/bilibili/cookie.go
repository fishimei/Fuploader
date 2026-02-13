package bilibili

import (
	"context"
	"fmt"
	"os"
	"strings"

	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

func (u *Uploader) ValidateCookie(ctx context.Context) (bool, error) {
	utils.InfoWithPlatform(u.platform, "验证Cookie")

	if _, err := os.Stat(u.cookiePath); os.IsNotExist(err) {
		return false, fmt.Errorf("cookie文件不存在")
	}

	browserCtx, err := u.browserPool.GetContextByAccount(ctx, u.accountID, u.cookiePath, nil)
	if err != nil {
		return false, fmt.Errorf("获取浏览器失败: %w", err)
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		return false, fmt.Errorf("获取页面失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "正在打开发布页面...")
	if _, err := page.Goto("https://member.bilibili.com/platform/upload/video/frame", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		return false, fmt.Errorf("打开页面失败: %w", err)
	}

	if err := page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State:   playwright.LoadStateNetworkidle,
		Timeout: playwright.Float(float64(u.config.PageLoadTimeout.Milliseconds())),
	}); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("等待页面加载超时: %v", err))
	}

	url := page.URL()
	if strings.Contains(url, "member.bilibili.com/platform/home") ||
		strings.Contains(url, "member.bilibili.com/platform/upload") {
		utils.SuccessWithPlatform(u.platform, "Cookie有效")
		return true, nil
	}

	utils.WarnWithPlatform(u.platform, "Cookie已失效")
	return false, nil
}
