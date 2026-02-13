package kuaishou

import (
	"context"
	"fmt"
	"os"

	"Fuploader/internal/platform/browser"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

func (u *Uploader) ValidateCookie(ctx context.Context) (bool, error) {
	utils.InfoWithPlatform(u.platform, "验证Cookie")

	if _, err := os.Stat(u.cookiePath); os.IsNotExist(err) {
		utils.WarnWithPlatform(u.platform, "Cookie文件不存在")
		return false, nil
	}

	browserCtx, err := u.getBrowserPool().GetContextByAccount(ctx, u.accountID, u.cookiePath, nil)
	if err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("获取浏览器失败: %v", err))
		return false, nil
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("获取页面失败: %v", err))
		return false, nil
	}

	if _, err := page.Goto("https://cp.kuaishou.com/", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("打开页面失败: %v", err))
		return false, nil
	}

	if err := page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State:   playwright.LoadStateNetworkidle,
		Timeout: playwright.Float(float64(u.config.PageLoadTimeout.Milliseconds())),
	}); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("等待页面加载超时: %v", err))
	}

	cookieConfig, ok := browser.GetCookieConfig("kuaishou")
	if !ok {
		return false, fmt.Errorf("获取Cookie配置失败")
	}

	isValid, err := browserCtx.ValidateLoginCookies(cookieConfig)
	if err != nil {
		return false, fmt.Errorf("验证失败: %v", err)
	}

	if isValid {
		utils.InfoWithPlatform(u.platform, fmt.Sprintf("检测到必需Cookie %v，验证通过", cookieConfig.RequiredCookies))
	} else {
		utils.InfoWithPlatform(u.platform, fmt.Sprintf("未检测到必需Cookie %v，验证失败", cookieConfig.RequiredCookies))
	}

	return isValid, nil
}
