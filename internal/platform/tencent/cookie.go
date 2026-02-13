package tencent

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
		utils.WarnWithPlatform(u.platform, "失败: 验证Cookie - Cookie文件不存在")
		return false, nil
	}

	browserCtx, err := u.getBrowserPool().GetContextByAccount(ctx, u.accountID, u.cookiePath, nil)
	if err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 验证Cookie - 获取浏览器上下文失败: %v", err))
		return false, nil
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 验证Cookie - 获取页面失败: %v", err))
		return false, nil
	}

	if _, err := page.Goto("https://channels.weixin.qq.com/platform/post/create", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 验证Cookie - 打开页面失败: %v", err))
		return false, nil
	}

	cookieConfig, ok := browser.GetCookieConfig("tencent")
	if !ok {
		return false, fmt.Errorf("失败: 验证Cookie - 获取视频号Cookie配置失败")
	}

	isValid, err := browserCtx.ValidateLoginCookies(cookieConfig)
	if err != nil {
		return false, fmt.Errorf("失败: 验证Cookie - 验证Cookie失败: %w", err)
	}

	if isValid {
		utils.InfoWithPlatform(u.platform, "登录成功")
	} else {
		utils.WarnWithPlatform(u.platform, "失败: 验证Cookie - Cookie无效或已过期")
	}

	return isValid, nil
}
