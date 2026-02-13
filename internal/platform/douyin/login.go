package douyin

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
	utils.InfoWithPlatform(u.platform, fmt.Sprintf("Cookie保存路径: %s", u.cookiePath))

	pool := u.getBrowserPool()
	browserCtx, err := pool.GetContext(ctx, u.cookiePath, nil)
	if err != nil {
		return fmt.Errorf("获取浏览器失败: %v", err)
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		return fmt.Errorf("获取页面失败: %v", err)
	}

	utils.InfoWithPlatform(u.platform, "正在打开发布页面...")
	if _, err := page.Goto("https://creator.douyin.com/creator-micro/content/upload", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		return fmt.Errorf("打开发布页面失败: %v", err)
	}

	utils.InfoWithPlatform(u.platform, "请在浏览器窗口中完成登录，登录成功后会自动保存")

	cookieConfig, ok := browser.GetCookieConfig("douyin")
	if !ok {
		return fmt.Errorf("获取Cookie配置失败")
	}

	if err := browserCtx.WaitForLoginCookies(cookieConfig); err != nil {
		return fmt.Errorf("等待登录Cookie失败: %v", err)
	}

	utils.SuccessWithPlatform(u.platform, "登录成功")
	if err := browserCtx.SaveCookiesTo(u.cookiePath); err != nil {
		return fmt.Errorf("保存Cookie失败: %v", err)
	}
	return nil
}
