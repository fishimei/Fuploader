package bilibili

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
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

	browserCtx, err := u.browserPool.GetContextByAccount(ctx, 0, "", nil)
	if err != nil {
		return fmt.Errorf("失败: 登录 - 获取浏览器失败: %w", err)
	}
	defer browserCtx.Release()

	page, err := browserCtx.GetPage()
	if err != nil {
		return fmt.Errorf("失败: 登录 - 获取页面失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "正在打开发布页面...")
	if _, err := page.Goto("https://member.bilibili.com/platform/upload/video/frame", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		return fmt.Errorf("失败: 登录 - 打开页面失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "请使用APP扫码登录")

	cookieConfig, ok := browser.GetCookieConfig("bilibili")
	if !ok {
		return fmt.Errorf("失败: 登录 - 获取Cookie配置失败")
	}

	if err := browserCtx.WaitForLoginCookies(cookieConfig); err != nil {
		return fmt.Errorf("失败: 登录 - 等待登录Cookie失败: %w", err)
	}

	loginSuccess := false
	for i := 0; i < u.config.MaxLoginWaitAttempts; i++ {
		url := page.URL()
		if strings.Contains(url, "member.bilibili.com/platform/home") ||
			strings.Contains(url, "member.bilibili.com/platform/upload") {
			loginSuccess = true
			break
		}
		time.Sleep(1 * time.Second)
		if i == u.config.MaxLoginWaitAttempts-1 {
			return fmt.Errorf("失败: 登录 - 等待跳转超时")
		}
	}

	if !loginSuccess {
		return fmt.Errorf("失败: 登录 - 登录验证失败")
	}

	utils.SuccessWithPlatform(u.platform, "登录成功")
	return u.saveCookiesFromPage(page)
}

func (u *Uploader) saveCookiesFromPage(page playwright.Page) error {
	debugLog("saveCookiesFromPage - cookiePath: '%s'", u.cookiePath)
	if u.cookiePath == "" {
		return fmt.Errorf("失败: 保存Cookie - cookie路径为空")
	}

	storageState, err := page.Context().StorageState()
	if err != nil {
		return fmt.Errorf("失败: 保存Cookie - 获取存储状态失败: %w", err)
	}

	data, err := json.Marshal(storageState)
	if err != nil {
		return fmt.Errorf("失败: 保存Cookie - 序列化失败: %w", err)
	}

	if err := os.WriteFile(u.cookiePath, data, 0644); err != nil {
		return fmt.Errorf("失败: 保存Cookie - 写入失败: %w", err)
	}

	return nil
}

func debugLog(format string, args ...interface{}) {
	if config.Config != nil && config.Config.DebugMode {
		utils.InfoWithPlatform("bilibili", fmt.Sprintf("[调试] "+format, args...))
	}
}
