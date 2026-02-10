package session

import (
	"context"
	"fmt"
	"time"

	"Fuploader/internal/platform/browser"
	"Fuploader/internal/types"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

// ValidationOptions 验证选项
type ValidationOptions struct {
	Timeout       time.Duration
	RetryCount    int
	RetryInterval time.Duration
	CheckAsync    bool // 是否检查异步加载内容
}

// DefaultValidationOptions 默认验证选项
func DefaultValidationOptions() ValidationOptions {
	return ValidationOptions{
		Timeout:       10 * time.Second,
		RetryCount:    3,
		RetryInterval: 2 * time.Second,
		CheckAsync:    true,
	}
}

// BaseValidator 基础验证器
type BaseValidator struct {
	platform string
	pool     *browser.Pool
	options  ValidationOptions
}

// Platform 返回平台名称
func (v *BaseValidator) Platform() string {
	return v.platform
}

// SetOptions 设置验证选项
func (v *BaseValidator) SetOptions(options ValidationOptions) {
	v.options = options
}

// validateWithCookie 使用Cookie检测进行验证
func (v *BaseValidator) validateWithCookie(
	ctx context.Context,
	session *Session,
	validateURL string,
) (bool, error) {
	for i := 0; i <= v.options.RetryCount; i++ {
		if i > 0 {
			utils.Info(fmt.Sprintf("[-] %s Cookie验证失败，第%d次重试...", v.platform, i))
			time.Sleep(v.options.RetryInterval)
		}

		// 使用浏览器池获取上下文
		browserCtx, err := v.pool.GetContext(ctx, v.getCookiePath(session), v.getContextOptions())
		if err != nil {
			if i == v.options.RetryCount {
				return false, types.NewNetworkError("validate", fmt.Errorf("get browser context failed: %w", err))
			}
			continue
		}
		defer browserCtx.Release()

		page, err := browserCtx.GetPage()
		if err != nil {
			if i == v.options.RetryCount {
				return false, types.NewNetworkError("validate", fmt.Errorf("get page failed: %w", err))
			}
			continue
		}

		// 访问验证页面
		if _, err := page.Goto(validateURL, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateNetworkidle,
			Timeout:   playwright.Float(10000),
		}); err != nil {
			if i == v.options.RetryCount {
				return false, types.NewTimeoutError("validate", err)
			}
			continue
		}

		// 等待页面加载
		time.Sleep(2 * time.Second)

		// 使用Cookie检测机制验证登录状态
		cookieConfig, ok := browser.GetCookieConfig(v.platform)
		if !ok {
			return false, fmt.Errorf("获取%sCookie配置失败", v.platform)
		}

		isValid, err := browserCtx.ValidateLoginCookies(cookieConfig)
		if err != nil {
			if i == v.options.RetryCount {
				return false, fmt.Errorf("验证Cookie失败: %w", err)
			}
			continue
		}

		return isValid, nil
	}

	return false, nil
}

func (v *BaseValidator) getCookiePath(session *Session) string {
	return fmt.Sprintf("cookies/%s_%d.json", session.Platform, session.AccountID)
}

func (v *BaseValidator) getContextOptions() *browser.ContextOptions {
	return &browser.ContextOptions{
		UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		Viewport:    &playwright.Size{Width: 1920, Height: 1080},
		Locale:      "zh-CN",
		TimezoneId:  "Asia/Shanghai",
		Geolocation: &playwright.Geolocation{Latitude: 39.9042, Longitude: 116.4074},
		ExtraHeaders: map[string]string{
			"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
		},
	}
}

// RegisterAllValidators 注册所有平台验证器
// TODO: 后续根据需求添加各平台验证器
func RegisterAllValidators(manager *Manager, pool *browser.Pool) {
	// 验证器暂时未启用，后续根据ValidateURL配置统一实现
	// manager.RegisterValidator(NewXiaoHongShuValidator(pool))
	// manager.RegisterValidator(NewDouyinValidator(pool))
	// manager.RegisterValidator(NewKuaishouValidator(pool))
	// manager.RegisterValidator(NewTencentValidator(pool))
	// manager.RegisterValidator(NewTikTokValidator(pool))
	// manager.RegisterValidator(NewBaijiahaoValidator(pool))
}
