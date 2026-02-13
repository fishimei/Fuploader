package xiaohongshu

import (
	"context"
	"fmt"
	"time"

	"Fuploader/internal/config"
	"Fuploader/internal/platform/browser"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

func (u *Uploader) Login() error {
	debugLog("Login开始 - cookiePath: '%s'", u.cookiePath)
	if u.cookiePath == "" {
		return fmt.Errorf("失败: 登录 - cookie路径为空")
	}

	ctx := context.Background()

	browserCtx, err := u.getBrowserPool().GetContextByAccount(ctx, 0, u.cookiePath, nil)
	if err != nil {
		return fmt.Errorf("失败: 登录 - 获取浏览器失败: %w", err)
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		return fmt.Errorf("失败: 登录 - 获取页面失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "正在打开发布页面...")
	if _, err := page.Goto("https://www.xiaohongshu.com", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 登录 - 访问主页失败: %v", err))
	} else {
		_, _ = page.Evaluate("window.scrollBy(0, 200)")
		time.Sleep(2 * time.Second)
	}

	utils.InfoWithPlatform(u.platform, "正在打开发布页面...")
	if _, err := page.Goto("https://creator.xiaohongshu.com/login", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		return fmt.Errorf("失败: 登录 - 打开登录页面失败: %w", err)
	}

	time.Sleep(3 * time.Second)

	utils.InfoWithPlatform(u.platform, "请在浏览器窗口中完成登录...")

	cookieConfig, ok := browser.GetCookieConfig("xiaohongshu")
	if !ok {
		return fmt.Errorf("失败: 登录 - 获取Cookie配置失败")
	}

	if err := browserCtx.WaitForLoginCookies(cookieConfig); err != nil {
		return fmt.Errorf("失败: 登录 - 等待登录Cookie失败: %w", err)
	}

	utils.SuccessWithPlatform(u.platform, "登录成功")
	if err := browserCtx.SaveCookiesTo(u.cookiePath); err != nil {
		return fmt.Errorf("失败: 登录 - 保存Cookie失败: %w", err)
	}
	return nil
}

func (u *Uploader) ValidateCookie(ctx context.Context) (bool, error) {
	utils.InfoWithPlatform(u.platform, "验证Cookie")

	browserCtx, err := u.getBrowserPool().GetContextByAccount(ctx, u.accountID, u.cookiePath, nil)
	if err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 验证Cookie - 获取浏览器失败: %v", err))
		return false, nil
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 验证Cookie - 获取页面失败: %v", err))
		return false, nil
	}

	utils.InfoWithPlatform(u.platform, "正在打开发布页面...")
	if _, err := page.Goto("https://creator.xiaohongshu.com/publish/publish?from=menu&target=video", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 验证Cookie - 打开页面失败: %v", err))
		return false, nil
	}

	time.Sleep(3 * time.Second)

	cookieConfig, ok := browser.GetCookieConfig("xiaohongshu")
	if !ok {
		return false, fmt.Errorf("失败: 验证Cookie - 获取Cookie配置失败")
	}

	isValid, err := browserCtx.ValidateLoginCookies(cookieConfig)
	if err != nil {
		return false, fmt.Errorf("失败: 验证Cookie - 验证失败: %w", err)
	}

	if isValid {
		utils.SuccessWithPlatform(u.platform, "登录成功")
	} else {
		utils.WarnWithPlatform(u.platform, "失败: 验证Cookie - 缺少必需Cookie")
	}

	return isValid, nil
}

func debugLog(format string, args ...interface{}) {
	if config.Config != nil && config.Config.DebugMode {
		utils.InfoWithPlatform("xiaohongshu", fmt.Sprintf("[调试] "+format, args...))
	}
}
