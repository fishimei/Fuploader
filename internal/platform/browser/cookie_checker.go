package browser

import (
	"context"
	"fmt"
	"strings"
	"time"

	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

// CookieChecker Cookie检测器
type CookieChecker struct {
	checkInterval time.Duration // 检测间隔
	timeout       time.Duration // 超时时间
}

// NewCookieChecker 创建Cookie检测器
func NewCookieChecker() *CookieChecker {
	return &CookieChecker{
		checkInterval: 2 * time.Second, // 检测间隔：2秒/次
		timeout:       5 * time.Minute, // 超时保护：5分钟
	}
}

// NewCookieCheckerWithTimeout 创建带自定义超时的Cookie检测器
func NewCookieCheckerWithTimeout(timeout time.Duration) *CookieChecker {
	return &CookieChecker{
		checkInterval: 2 * time.Second,
		timeout:       timeout,
	}
}

// WaitForLoginCookies 等待登录Cookie出现
// 循环检测：「全量获取→映射判空→全量满足即返回」
func (cc *CookieChecker) WaitForLoginCookies(
	ctx context.Context,
	page playwright.Page,
	config PlatformCookieConfig,
) error {
	timeout := time.After(cc.timeout)
	ticker := time.NewTicker(cc.checkInterval)
	defer ticker.Stop()

	// 获取需要检测的域名列表
	domains := cc.getCheckDomains(config)
	utils.Info(fmt.Sprintf("[-] 开始检测登录Cookie，目标域名: %v，必需字段: %v",
		domains, config.GetAllCookies()))

	checkCount := 0
	for {
		select {
		case <-timeout:
			return fmt.Errorf("登录Cookie检测超时（%v），未检测到必需Cookie",
				cc.timeout)
		case <-ctx.Done():
			return fmt.Errorf("context取消: %w", ctx.Err())
		case <-ticker.C:
			checkCount++
			// 检测页面是否已关闭
			if page == nil {
				return fmt.Errorf("页面已关闭")
			}

			// 多域名检测
			allValid := true
			for _, domainConfig := range domains {
				valid, err := cc.checkDomainCookies(page, domainConfig, checkCount)
				if err != nil {
					if isBrowserClosedError(err) {
						return fmt.Errorf("浏览器已关闭，终止Cookie检测: %w", err)
					}
					utils.Warn(fmt.Sprintf("[-] 检测域名 %s Cookie失败: %v", domainConfig.Domain, err))
					allValid = false
					break
				}
				if !valid {
					allValid = false
					break
				}
			}

			// 所有域名都满足条件 → 检测通过
			if allValid {
				utils.Info(fmt.Sprintf("[-] 检测到所有必需Cookie"))
				return nil
			}
		}
	}
}

// getCheckDomains 获取需要检测的域名配置列表
func (cc *CookieChecker) getCheckDomains(config PlatformCookieConfig) []CookieDomainConfig {
	if len(config.Domains) > 0 {
		return config.Domains
	}
	// 兼容旧版单域名配置
	return []CookieDomainConfig{
		{
			Domain:          "", // 使用当前页面域名
			RequiredCookies: config.RequiredCookies,
			ExtendedCookies: config.ExtendedCookies,
		},
	}
}

// checkDomainCookies 检测单个域名的Cookie
func (cc *CookieChecker) checkDomainCookies(
	page playwright.Page,
	config CookieDomainConfig,
	checkCount int,
) (bool, error) {
	domainStr := config.Domain
	if domainStr == "" {
		domainStr = "当前页面"
	}

	// 全量获取Cookie
	// 如果Domain为空，则获取所有Cookie（不传域名参数）
	var cookies []playwright.Cookie
	var err error
	if config.Domain == "" {
		cookies, err = page.Context().Cookies()
	} else {
		cookies, err = page.Context().Cookies(config.Domain)
	}
	if err != nil {
		utils.Warn(fmt.Sprintf("[-] 获取域名 [%s] Cookie失败: %v", domainStr, err))
		return false, err
	}

	utils.Info(fmt.Sprintf("[-] 域名 [%s] 获取到 %d 个Cookie", domainStr, len(cookies)))

	// 转为map[name]value键值对，方便快速查询（同时创建大小写不敏感的映射）
	cookieMap := make(map[string]string, len(cookies))
	cookieMapLower := make(map[string]string, len(cookies))
	for _, cookie := range cookies {
		cookieMap[cookie.Name] = cookie.Value
		cookieMapLower[strings.ToLower(cookie.Name)] = cookie.Value
	}

	// 每次检测都输出调试信息
	cookieNames := make([]string, 0, len(cookieMap))
	for name := range cookieMap {
		cookieNames = append(cookieNames, name)
	}
	utils.Info(fmt.Sprintf("[-] 域名 [%s] 所有Cookie名称: %v", domainStr, cookieNames))

	// 显示必需Cookie状态
	utils.Info(fmt.Sprintf("[-] 域名 [%s] 必需Cookie检测:", domainStr))
	allRequiredExist := true
	for _, name := range config.RequiredCookies {
		if value, exists := cookieMap[name]; exists {
			utils.Info(fmt.Sprintf("    ✓ %s: 存在 (值长度=%d)", name, len(value)))
		} else if value, exists := cookieMapLower[strings.ToLower(name)]; exists {
			utils.Info(fmt.Sprintf("    ✓ %s: 存在 (大小写不同,值长度=%d)", name, len(value)))
		} else {
			utils.Info(fmt.Sprintf("    ✗ %s: 不存在", name))
			allRequiredExist = false
		}
	}

	// 显示扩展Cookie状态
	if len(config.ExtendedCookies) > 0 {
		utils.Info(fmt.Sprintf("[-] 域名 [%s] 扩展Cookie检测:", domainStr))
		for _, name := range config.ExtendedCookies {
			if value, exists := cookieMap[name]; exists {
				utils.Info(fmt.Sprintf("    ✓ %s: 存在 (值长度=%d)", name, len(value)))
			} else if value, exists := cookieMapLower[strings.ToLower(name)]; exists {
				utils.Info(fmt.Sprintf("    ✓ %s: 存在 (大小写不同,值长度=%d)", name, len(value)))
			} else {
				utils.Info(fmt.Sprintf("    ✗ %s: 不存在", name))
			}
		}
	}

	// 返回必需Cookie是否全部存在
	if allRequiredExist {
		utils.Info(fmt.Sprintf("[-] 域名 [%s] 所有必需Cookie已检测到", domainStr))
	} else {
		utils.Info(fmt.Sprintf("[-] 域名 [%s] 缺少必需Cookie", domainStr))
	}

	return allRequiredExist, nil
}

// ValidateLoginCookies 验证当前Cookie是否有效（单次检测）
func (cc *CookieChecker) ValidateLoginCookies(
	page playwright.Page,
	config PlatformCookieConfig,
) (bool, error) {
	if page == nil {
		return false, fmt.Errorf("页面为空")
	}

	// 获取需要检测的域名列表
	domains := cc.getCheckDomains(config)

	// 多域名检测
	for _, domainConfig := range domains {
		valid, err := cc.checkDomainCookies(page, domainConfig, 0)
		if err != nil {
			return false, fmt.Errorf("验证域名 %s Cookie失败: %w", domainConfig.Domain, err)
		}
		if !valid {
			return false, nil
		}
	}

	return true, nil
}

// checkPageElements 检查页面元素确认登录状态
// 通过检查上传页面特有的元素来判断是否登录成功
func (cc *CookieChecker) checkPageElements(page playwright.Page) error {
	// 检查用户头像或上传区域等登录后才有的元素
	// B站上传页登录后会有文件上传输入框
	selectors := []string{
		`input[type="file"][accept*="video"]`, // 视频上传输入框
		`.bcc-upload-wrapper`,                 // 上传区域
		`.user-avatar`,                        // 用户头像
		`.creator-header`,                     // 创作者头部
	}

	for _, selector := range selectors {
		element := page.Locator(selector).First()
		if count, _ := element.Count(); count > 0 {
			utils.Info(fmt.Sprintf("[-] 页面元素检查 - 找到元素: %s", selector))
			return nil
		}
	}

	return fmt.Errorf("未找到登录后的页面元素")
}

// isBrowserClosedError 判断错误是否由浏览器关闭引起
func isBrowserClosedError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "target closed") ||
		strings.Contains(errMsg, "browser has been closed") ||
		strings.Contains(errMsg, "context or browser has been closed") ||
		strings.Contains(errMsg, "page has been closed") ||
		strings.Contains(errMsg, "connection closed")
}

// GetCookieValues 获取指定Cookie的值
func (cc *CookieChecker) GetCookieValues(
	page playwright.Page,
	domain string,
	names []string,
) (map[string]string, error) {
	cookies, err := page.Context().Cookies(domain)
	if err != nil {
		return nil, fmt.Errorf("获取Cookie失败: %w", err)
	}

	result := make(map[string]string)
	cookieMap := make(map[string]string, len(cookies))
	for _, cookie := range cookies {
		cookieMap[cookie.Name] = cookie.Value
	}

	for _, name := range names {
		if value, exists := cookieMap[name]; exists {
			result[name] = value
		}
	}

	return result, nil
}
