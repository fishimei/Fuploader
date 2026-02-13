package baijiahao

import (
	"context"
	"fmt"
	"os"
	"time"

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
	utils.InfoWithPlatform(u.platform, "正在打开登录页面...")

	browserCtx, err := u.getBrowserPool().GetContextByAccount(ctx, 0, "", nil)
	if err != nil {
		return fmt.Errorf("失败: 登录 - 获取浏览器失败: %w", err)
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		return fmt.Errorf("失败: 登录 - 获取页面失败: %w", err)
	}

	if _, err := page.Goto("https://baijiahao.baidu.com/builder/theme/bjh/login", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		return fmt.Errorf("失败: 登录 - 打开登录页面失败: %w", err)
	}

	time.Sleep(3 * time.Second)

	utils.InfoWithPlatform(u.platform, "请在浏览器窗口中完成登录...")

	cookieConfig, ok := browser.GetCookieConfig("baijiahao")
	if !ok {
		return fmt.Errorf("失败: 登录 - 获取Cookie配置失败")
	}

	if err := browserCtx.WaitForLoginCookies(cookieConfig); err != nil {
		return fmt.Errorf("失败: 登录 - 等待Cookie失败: %w", err)
	}

	utils.SuccessWithPlatform(u.platform, "登录成功")
	if err := browserCtx.SaveCookiesTo(u.cookiePath); err != nil {
		return fmt.Errorf("失败: 登录 - 保存Cookie失败: %w", err)
	}
	return nil
}

func (u *Uploader) ValidateCookie(ctx context.Context) (bool, error) {
	utils.InfoWithPlatform(u.platform, "验证Cookie")

	if _, err := os.Stat(u.cookiePath); os.IsNotExist(err) {
		utils.WarnWithPlatform(u.platform, "失败: 验证Cookie - Cookie文件不存在")
		return false, nil
	}

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

	if _, err := page.Goto("https://baijiahao.baidu.com/builder/rc/home", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 验证Cookie - 打开页面失败: %v", err))
		return false, nil
	}

	time.Sleep(5 * time.Second)

	cookieConfig, ok := browser.GetCookieConfig("baijiahao")
	if !ok {
		return false, fmt.Errorf("失败: 验证Cookie - 获取Cookie配置失败")
	}

	isValid, err := browserCtx.ValidateLoginCookies(cookieConfig)
	if err != nil {
		return false, fmt.Errorf("失败: 验证Cookie - %w", err)
	}

	if isValid {
		utils.InfoWithPlatform(u.platform, "检测到PTOKEN和BAIDUID Cookie，验证通过")
	} else {
		utils.InfoWithPlatform(u.platform, "未检测到PTOKEN和BAIDUID Cookie，验证失败")
	}

	return isValid, nil
}
