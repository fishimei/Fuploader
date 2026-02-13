package tiktok

import (
	"context"
	"fmt"

	"Fuploader/internal/platform/browser"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

func (u *Uploader) Login() error {
	debugLog("Login开始 - cookiePath: '%s'", u.cookiePath)
	if u.cookiePath == "" {
		return fmt.Errorf("cookie路径为空")
	}

	ctx := context.Background()
	pool := u.getBrowserPool()

	browserCtx, err := pool.GetContextByAccount(ctx, 0, "", u.getContextOptions())
	if err != nil {
		return fmt.Errorf("获取浏览器失败: %w", err)
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		return fmt.Errorf("获取页面失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "正在打开登录页面...")
	if _, err := page.Goto("https://www.tiktok.com/login?lang=en", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		return fmt.Errorf("打开登录页面失败: %w", err)
	}

	cookieConfig, ok := browser.GetCookieConfig("tiktok")
	if !ok {
		return fmt.Errorf("获取TikTok Cookie配置失败")
	}

	if err := browserCtx.WaitForLoginCookies(cookieConfig); err != nil {
		return fmt.Errorf("等待登录Cookie失败: %w", err)
	}

	utils.SuccessWithPlatform(u.platform, "登录成功")
	if err := browserCtx.SaveCookiesTo(u.cookiePath); err != nil {
		return fmt.Errorf("保存Cookie失败: %w", err)
	}
	return nil
}
