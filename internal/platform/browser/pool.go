package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"Fuploader/internal/config"
	"Fuploader/internal/platform/platformutils"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

// PoolStats 浏览器池统计信息
type PoolStats struct {
	BrowserCount      int       `json:"browser_count"`        // 当前浏览器实例数
	ContextCount      int       `json:"context_count"`        // 当前上下文总数
	IdleContextCount  int       `json:"idle_context_count"`   // 空闲上下文数
	InUseContextCount int       `json:"in_use_context_count"` // 使用中上下文数
	WaitQueueLength   int       `json:"wait_queue_length"`    // 等待队列长度
	MaxBrowsers       int       `json:"max_browsers"`         // 最大浏览器数
	MaxContexts       int       `json:"max_contexts"`         // 每个浏览器的最大上下文数
	Timestamp         time.Time `json:"timestamp"`            // 统计时间戳
}

// Pool 浏览器池
type Pool struct {
	maxBrowsers int
	maxContexts int
	browsers    []*PooledBrowser
	mutex       sync.RWMutex
	waitQueue   chan struct{} // 等待队列，用于限制并发获取上下文
	stats       PoolStats
	statsMutex  sync.RWMutex
}

// PooledBrowser 池化浏览器
type PooledBrowser struct {
	browser  playwright.Browser
	contexts []*PooledContext
	lastUsed time.Time
	inUse    int
	mutex    sync.Mutex
}

// PooledContext 封装的浏览器上下文
type PooledContext struct {
	context    playwright.BrowserContext
	page       playwright.Page
	cookiePath string
	createdAt  time.Time
	lastUsed   time.Time
	parent     *PooledBrowser
	platform   string // 平台标识，用于日志
}

// ContextOptions 上下文选项
type ContextOptions struct {
	UserAgent    string
	Viewport     *playwright.Size
	Locale       string
	TimezoneId   string
	Geolocation  *playwright.Geolocation
	ExtraHeaders map[string]string
	// 反爬相关选项
	EnableAntiDetect  bool // 启用反检测
	EnableRandomDelay bool // 启用随机延迟
	HumanLikeBehavior bool // 模拟人类行为
}

// DefaultContextOptions 返回默认上下文选项（带反爬配置）
func DefaultContextOptions() *ContextOptions {
	return &ContextOptions{
		EnableAntiDetect:  true,
		EnableRandomDelay: true,
		HumanLikeBehavior: true,
	}
}

// NewPool 创建浏览器池
func NewPool(maxBrowsers, maxContexts int) *Pool {
	return &Pool{
		maxBrowsers: maxBrowsers,
		maxContexts: maxContexts,
		browsers:    make([]*PooledBrowser, 0),
		waitQueue:   make(chan struct{}, maxBrowsers*maxContexts), // 限制并发数
	}
}

// NewPoolFromConfig 从配置创建浏览器池
func NewPoolFromConfig() *Pool {
	poolConfig := LoadPoolConfig()
	return NewPool(poolConfig.MaxBrowsers, poolConfig.MaxContextsPerBrowser)
}

// GetContext 获取浏览器上下文
func (p *Pool) GetContext(ctx context.Context, cookiePath string, options *ContextOptions) (*PooledContext, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 使用默认选项
	if options == nil {
		options = DefaultContextOptions()
	}

	// 如果启用反检测，生成随机指纹
	if options.EnableAntiDetect {
		options = p.generateRandomFingerprint(options)
	}

	// 1. 尝试复用现有上下文
	for _, browser := range p.browsers {
		if pooledCtx := browser.getIdleContext(cookiePath); pooledCtx != nil {
			p.updateStats()
			return pooledCtx, nil
		}
	}

	// 2. 创建新上下文
	browser, err := p.getOrCreateBrowser()
	if err != nil {
		return nil, err
	}

	pooledCtx, err := browser.createContext(cookiePath, options)
	if err != nil {
		return nil, err
	}

	p.updateStats()
	return pooledCtx, nil
}

// generateRandomFingerprint 生成随机浏览器指纹
func (p *Pool) generateRandomFingerprint(baseOptions *ContextOptions) *ContextOptions {
	chromeVersions := []string{"120", "121", "122", "123", "124", "125"}
	version := chromeVersions[rand.Intn(len(chromeVersions))]

	// 随机视口（在合理范围内变化）
	width := 1920 + rand.Intn(100) - 50
	height := 1080 + rand.Intn(100) - 50

	// 随机地理位置（北京附近）
	lat := 39.9042 + (rand.Float64()-0.5)*0.1
	lng := 116.4074 + (rand.Float64()-0.5)*0.1

	options := &ContextOptions{
		UserAgent: fmt.Sprintf(
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s.0.0.0 Safari/537.36",
			version,
		),
		Viewport: &playwright.Size{
			Width:  width,
			Height: height,
		},
		Locale:     "zh-CN",
		TimezoneId: "Asia/Shanghai",
		Geolocation: &playwright.Geolocation{
			Latitude:  lat,
			Longitude: lng,
		},
		ExtraHeaders: map[string]string{
			"Accept-Language":           "zh-CN,zh;q=0.9,en;q=0.8",
			"Sec-Ch-Ua":                 fmt.Sprintf(`"Not_A Brand";v="8", "Chromium";v="%s", "Google Chrome";v="%s"`, version, version),
			"Sec-Ch-Ua-Mobile":          "?0",
			"Sec-Ch-Ua-Platform":        `"Windows"`,
			"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
			"Accept-Encoding":           "gzip, deflate, br",
			"Upgrade-Insecure-Requests": "1",
		},
		EnableAntiDetect:  baseOptions.EnableAntiDetect,
		EnableRandomDelay: baseOptions.EnableRandomDelay,
		HumanLikeBehavior: baseOptions.HumanLikeBehavior,
	}

	return options
}

// GetStats 获取浏览器池统计信息
func (p *Pool) GetStats() PoolStats {
	p.statsMutex.RLock()
	defer p.statsMutex.RUnlock()
	return p.stats
}

// updateStats 更新统计信息
func (p *Pool) updateStats() {
	p.statsMutex.Lock()
	defer p.statsMutex.Unlock()

	p.stats = PoolStats{
		BrowserCount: len(p.browsers),
		MaxBrowsers:  p.maxBrowsers,
		MaxContexts:  p.maxContexts,
		Timestamp:    time.Now(),
	}

	for _, browser := range p.browsers {
		browser.mutex.Lock()
		p.stats.ContextCount += len(browser.contexts)
		p.stats.InUseContextCount += browser.inUse
		for _, ctx := range browser.contexts {
			if time.Since(ctx.lastUsed) > 30*time.Second {
				p.stats.IdleContextCount++
			}
		}
		browser.mutex.Unlock()
	}

	p.stats.WaitQueueLength = len(p.waitQueue)
}

// Close 关闭浏览器池
func (p *Pool) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for _, browser := range p.browsers {
		for _, ctx := range browser.contexts {
			ctx.Close()
		}
		if err := browser.browser.Close(); err != nil {
			utils.Warn(fmt.Sprintf("[-] 关闭浏览器失败: %v", err))
		}
	}

	p.browsers = make([]*PooledBrowser, 0)
	p.updateStats()
	return nil
}

// getOrCreateBrowser 获取或创建浏览器实例
func (p *Pool) getOrCreateBrowser() (*PooledBrowser, error) {
	// 查找有可用容量的浏览器
	for _, b := range p.browsers {
		if b.canCreateContext(p.maxContexts) {
			return b, nil
		}
	}

	// 创建新浏览器
	if len(p.browsers) < p.maxBrowsers {
		browser, err := p.launchBrowser()
		if err != nil {
			return nil, err
		}

		pooled := &PooledBrowser{
			browser:  browser,
			contexts: make([]*PooledContext, 0),
		}
		p.browsers = append(p.browsers, pooled)
		return pooled, nil
	}

	return nil, fmt.Errorf("max browsers reached")
}

// launchBrowser 启动浏览器
func (p *Pool) launchBrowser() (playwright.Browser, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("start playwright failed: %w", err)
	}

	// 查找本地 Chrome
	chromePath := findLocalChrome()

	launchOptions := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(config.Config.Headless),
		Args: []string{
			"--disable-blink-features=AutomationControlled",
			"--disable-web-security",
			"--no-sandbox",
			"--disable-setuid-sandbox",
			"--disable-dev-shm-usage",
			"--window-size=1920,1080",
			"--window-position=0,0",
			"--start-maximized",
			"--disable-infobars",
			"--disable-extensions",
			"--disable-default-apps",
			"--disable-background-networking",
			"--disable-sync",
			"--disable-translate",
			"--disable-popup-blocking",
			"--disable-features=IsolateOrigins,site-per-process,SameSiteByDefaultCookies,CookiesWithoutSameSiteMustBeSecure",
			"--disable-site-isolation-trials",
		},
	}

	if chromePath != "" {
		launchOptions.ExecutablePath = playwright.String(chromePath)
		utils.Info("[-] 浏览器池使用本地 Chrome")
	}

	browser, err := pw.Chromium.Launch(launchOptions)
	if err != nil {
		return nil, fmt.Errorf("launch browser failed: %w", err)
	}

	return browser, nil
}

// canCreateContext 检查是否可以创建新上下文
func (b *PooledBrowser) canCreateContext(maxContexts int) bool {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return len(b.contexts) < maxContexts
}

// getIdleContext 获取空闲上下文
func (b *PooledBrowser) getIdleContext(cookiePath string) *PooledContext {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	for _, ctx := range b.contexts {
		if ctx.cookiePath == cookiePath && time.Since(ctx.lastUsed) > 30*time.Second {
			ctx.lastUsed = time.Now()
			b.inUse++
			return ctx
		}
	}
	return nil
}

// createContext 创建浏览器上下文
func (b *PooledBrowser) createContext(cookiePath string, options *ContextOptions) (*PooledContext, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	contextOptions := playwright.BrowserNewContextOptions{
		Locale:           playwright.String(options.Locale),
		TimezoneId:       playwright.String(options.TimezoneId),
		Permissions:      []string{"geolocation"},
		ColorScheme:      playwright.ColorSchemeLight,
		ExtraHttpHeaders: options.ExtraHeaders,
	}

	if options.UserAgent != "" {
		contextOptions.UserAgent = playwright.String(options.UserAgent)
	}
	if options.Geolocation != nil {
		contextOptions.Geolocation = options.Geolocation
	}

	// 加载 Cookie
	if _, err := os.Stat(cookiePath); err == nil {
		contextOptions.StorageStatePath = playwright.String(cookiePath)
	}

	context, err := b.browser.NewContext(contextOptions)
	if err != nil {
		return nil, fmt.Errorf("create context failed: %w", err)
	}

	// 注入反检测脚本
	if err := platformutils.InjectStealthScript(context); err != nil {
		return nil, fmt.Errorf("inject stealth script failed: %w", err)
	}

	ctx := &PooledContext{
		context:    context,
		cookiePath: cookiePath,
		createdAt:  time.Now(),
		lastUsed:   time.Now(),
		parent:     b,
	}

	b.contexts = append(b.contexts, ctx)
	b.inUse++

	return ctx, nil
}

// Release 释放上下文
func (c *PooledContext) Release() error {
	c.parent.mutex.Lock()
	defer c.parent.mutex.Unlock()

	// 获取平台标识，如果为空则使用默认值
	platform := c.platform
	if platform == "" {
		platform = "browser"
	}

	// 检查页面是否已关闭（用户手动关闭浏览器）
	if c.IsPageClosed() {
		utils.Info(fmt.Sprintf("[-] [%s] 浏览器被用户关闭，执行清理...", platform))

		// 尝试保存Cookie（如果可能）
		if c.cookiePath != "" {
			utils.Info(fmt.Sprintf("[-] [%s] 尝试保存Cookie状态...", platform))
			if err := c.SaveCookiesTo(c.cookiePath); err != nil {
				utils.Warn(fmt.Sprintf("[-] [%s] 保存Cookie失败（页面已关闭）: %v", platform, err))
			} else {
				utils.Info(fmt.Sprintf("[-] [%s] Cookie已保存", platform))
			}
		}

		// 关闭整个上下文
		if err := c.context.Close(); err != nil {
			utils.Warn(fmt.Sprintf("[-] [%s] 关闭上下文失败: %v", platform, err))
		}

		// 从父浏览器的上下文中移除
		c.removeFromParent()
		c.parent.inUse--

		utils.Info(fmt.Sprintf("[-] [%s] 浏览器上下文已清理完成", platform))
		return fmt.Errorf("browser was closed by user")
	}

	// 正常释放流程（页面未关闭）
	utils.Info(fmt.Sprintf("[-] [%s] 释放浏览器上下文...", platform))

	// 保存 Cookie
	if err := c.saveCookie(); err != nil {
		utils.Warn(fmt.Sprintf("[-] [%s] 保存Cookie失败: %v", platform, err))
	} else {
		utils.Info(fmt.Sprintf("[-] [%s] Cookie已保存", platform))
	}

	// 关闭页面
	if c.page != nil {
		utils.Info(fmt.Sprintf("[-] [%s] 关闭浏览器页面...", platform))
		if err := c.page.Close(); err != nil {
			utils.Warn(fmt.Sprintf("[-] [%s] 关闭页面失败: %v", platform, err))
		} else {
			utils.Info(fmt.Sprintf("[-] [%s] 浏览器页面已关闭", platform))
		}
		c.page = nil
	}

	c.parent.inUse--
	c.lastUsed = time.Now()

	utils.Info(fmt.Sprintf("[-] [%s] 浏览器上下文已释放", platform))

	return nil
}

// removeFromParent 从父浏览器中移除上下文
func (c *PooledContext) removeFromParent() {
	for i, ctx := range c.parent.contexts {
		if ctx == c {
			// 从切片中移除
			c.parent.contexts = append(c.parent.contexts[:i], c.parent.contexts[i+1:]...)
			break
		}
	}
}

// saveCookie 保存 Cookie（私有方法）
func (c *PooledContext) saveCookie() error {
	return c.SaveCookies()
}

// SaveCookies 保存 Cookie（公共方法，供外部调用）
func (c *PooledContext) SaveCookies() error {
	if c.cookiePath == "" {
		return fmt.Errorf("cookie path is empty")
	}
	return c.SaveCookiesTo(c.cookiePath)
}

// SaveCookiesTo 保存 Cookie 到指定路径
func (c *PooledContext) SaveCookiesTo(cookiePath string) error {
	storage, err := c.context.StorageState()
	if err != nil {
		return err
	}

	data, err := json.Marshal(storage)
	if err != nil {
		return err
	}

	// 确保目录存在
	dir := filepath.Dir(cookiePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create cookie directory failed: %w", err)
		}
	}

	return os.WriteFile(cookiePath, data, 0644)
}

// GetPage 获取或创建页面
func (c *PooledContext) GetPage() (playwright.Page, error) {
	if c.page != nil {
		return c.page, nil
	}

	utils.Info(fmt.Sprintf("[-] [%s] 创建浏览器页面...", c.platform))

	page, err := c.context.NewPage()
	if err != nil {
		return nil, err
	}

	// 设置默认超时
	page.SetDefaultTimeout(30000) // 30秒
	page.SetDefaultNavigationTimeout(30000)

	// 监听页面关闭事件
	page.On("close", func() {
		utils.Info(fmt.Sprintf("[-] [%s] 浏览器页面被关闭（用户操作或系统）", c.platform))
		c.page = nil
	})

	c.page = page
	utils.Info(fmt.Sprintf("[-] [%s] 浏览器页面创建成功", c.platform))
	return page, nil
}

// WaitForPageLoad 等待页面完全加载
func (c *PooledContext) WaitForPageLoad() error {
	if c.page == nil {
		return fmt.Errorf("page not created")
	}
	// 等待网络空闲，确保所有资源加载完成
	return c.page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	})
}

// IsPageClosed 检查页面是否已关闭
func (c *PooledContext) IsPageClosed() bool {
	if c.page == nil {
		return true
	}

	// 重试3次，每次间隔500ms
	maxRetries := 3
	retryDelay := 500 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		// 如果本次检测通过，直接返回未关闭
		if c.checkPageAlive() {
			return false
		}

		// 如果不是最后一次，等待后重试
		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}

	// 连续3次检测失败，判定为页面已关闭
	return true
}

// checkPageAlive 检查页面是否存活（单次检测）
func (c *PooledContext) checkPageAlive() bool {
	// 方法1: 尝试执行简单的 JS
	_, err := c.page.Evaluate("1")
	if err != nil {
		return false
	}

	// 方法2: 检查页面 URL
	_, err = c.page.Evaluate(`window.location.href`)
	if err != nil {
		return false
	}

	// 方法3: 检查页面标题
	_, err = c.page.Evaluate(`document.title`)
	if err != nil {
		return false
	}

	return true
}

// Close 关闭上下文（强制关闭）
func (c *PooledContext) Close() error {
	utils.Info(fmt.Sprintf("[-] [%s] 强制关闭浏览器上下文...", c.platform))

	if c.page != nil {
		utils.Info(fmt.Sprintf("[-] [%s] 关闭浏览器页面...", c.platform))
		if err := c.page.Close(); err != nil {
			utils.Warn(fmt.Sprintf("[-] [%s] 关闭页面失败: %v", c.platform, err))
		} else {
			utils.Info(fmt.Sprintf("[-] [%s] 浏览器页面已关闭", c.platform))
		}
		c.page = nil
	}

	if err := c.context.Close(); err != nil {
		utils.Warn(fmt.Sprintf("[-] [%s] 关闭上下文失败: %v", c.platform, err))
		return err
	}

	utils.Info(fmt.Sprintf("[-] [%s] 浏览器上下文已关闭", c.platform))
	return nil
}

// ClosePage 关闭页面（上传成功后调用）
func (c *PooledContext) ClosePage() error {
	if c.page != nil {
		utils.Info(fmt.Sprintf("[-] [%s] 关闭浏览器页面...", c.platform))
		if err := c.page.Close(); err != nil {
			utils.Warn(fmt.Sprintf("[-] [%s] 关闭页面失败: %v", c.platform, err))
			return err
		}
		c.page = nil
		utils.Info(fmt.Sprintf("[-] [%s] 浏览器页面已关闭", c.platform))
	}
	return nil
}

// ==================== Cookie检测方法 ====================

// WaitForLoginCookies 等待登录Cookie出现
// 循环检测：「全量获取→映射判空→全量满足即返回」
func (c *PooledContext) WaitForLoginCookies(config PlatformCookieConfig) error {
	if c.page == nil {
		return fmt.Errorf("page not created")
	}

	checker := NewCookieChecker()
	return checker.WaitForLoginCookies(context.Background(), c.page, config)
}

// WaitForLoginCookiesWithContext 带context的等待登录Cookie
func (c *PooledContext) WaitForLoginCookiesWithContext(ctx context.Context, config PlatformCookieConfig) error {
	if c.page == nil {
		return fmt.Errorf("page not created")
	}

	checker := NewCookieChecker()
	return checker.WaitForLoginCookies(ctx, c.page, config)
}

// ValidateLoginCookies 验证当前Cookie是否有效
func (c *PooledContext) ValidateLoginCookies(config PlatformCookieConfig) (bool, error) {
	if c.page == nil {
		return false, fmt.Errorf("page not created")
	}

	checker := NewCookieChecker()
	return checker.ValidateLoginCookies(c.page, config)
}

// GetCookieValues 获取指定Cookie的值
func (c *PooledContext) GetCookieValues(domain string, names []string) (map[string]string, error) {
	if c.page == nil {
		return nil, fmt.Errorf("page not created")
	}

	checker := NewCookieChecker()
	return checker.GetCookieValues(c.page, domain, names)
}

// ==================== 反爬检测方法 ====================

// DetectCaptcha 检测是否出现验证码/滑块
func (c *PooledContext) DetectCaptcha() (bool, string, error) {
	if c.page == nil {
		return false, "", fmt.Errorf("page not created")
	}

	captchaSelectors := []struct {
		selector string
		type_    string
	}{
		{".captcha", "验证码"},
		{"[class*='captcha']", "验证码"},
		{"[class*='slider']", "滑块验证"},
		{"[class*='verify']", "验证"},
		{".geetest", "极验验证"},
		{"[class*='geetest']", "极验验证"},
		{"iframe[src*='captcha']", "验证码iframe"},
		{"iframe[src*='verify']", "验证iframe"},
		{"text=请完成验证", "文字验证"},
		{"text=拖动滑块", "滑块验证"},
		{"text=点击验证", "点击验证"},
	}

	for _, item := range captchaSelectors {
		count, err := c.page.Locator(item.selector).Count()
		if err == nil && count > 0 {
			visible, _ := c.page.Locator(item.selector).IsVisible()
			if visible {
				utils.Warn(fmt.Sprintf("[-] 检测到%s", item.type_))
				return true, item.type_, nil
			}
		}
	}

	verificationTexts := []string{
		"请完成安全验证",
		"请进行验证",
		"验证失败",
		"请点击",
		"请拖动",
	}

	for _, text := range verificationTexts {
		count, _ := c.page.GetByText(text).Count()
		if count > 0 {
			return true, "验证提示", nil
		}
	}

	return false, "", nil
}

// DetectAntiBot 检测反爬虫标记
func (c *PooledContext) DetectAntiBot() (bool, string, error) {
	if c.page == nil {
		return false, "", fmt.Errorf("page not created")
	}

	antiBotIndicators := []struct {
		selector string
		message  string
	}{
		{"text=访问过于频繁", "访问频繁"},
		{"text=操作过于频繁", "操作频繁"},
		{"text=请稍后再试", "限流提示"},
		{"text=系统繁忙", "系统繁忙"},
		{"text=网络异常", "网络异常"},
		{"text=账号异常", "账号异常"},
		{"text=登录异常", "登录异常"},
		{"text=自动程序", "自动程序检测"},
		{"text=机器人", "机器人检测"},
		{"[class*='ban']", "封禁提示"},
		{"[class*='block']", "拦截提示"},
	}

	for _, item := range antiBotIndicators {
		count, err := c.page.Locator(item.selector).Count()
		if err == nil && count > 0 {
			visible, _ := c.page.Locator(item.selector).IsVisible()
			if visible {
				utils.Warn(fmt.Sprintf("[-] 检测到反爬标记: %s", item.message))
				return true, item.message, nil
			}
		}
	}

	return false, "", nil
}

// ==================== 人类行为模拟方法 ====================

// HumanLikeDelay 模拟人类操作的随机延迟
func (c *PooledContext) HumanLikeDelay(baseDelay time.Duration) {
	variance := float64(baseDelay) * 0.3
	delay := baseDelay + time.Duration(rand.Float64()*variance*2-variance)
	time.Sleep(delay)
}

// HumanLikeTyping 模拟人类输入（带随机延迟）
func (c *PooledContext) HumanLikeTyping(text string) error {
	if c.page == nil {
		return fmt.Errorf("page not created")
	}

	for _, char := range text {
		if err := c.page.Keyboard().Type(string(char)); err != nil {
			return err
		}
		time.Sleep(time.Duration(50+rand.Intn(100)) * time.Millisecond)
	}
	return nil
}

// SimulateHumanBehavior 模拟人类浏览行为
func (c *PooledContext) SimulateHumanBehavior() error {
	if c.page == nil {
		return fmt.Errorf("page not created")
	}

	// 随机滚动
	scrollCount := 2 + rand.Intn(3)
	for i := 0; i < scrollCount; i++ {
		scrollY := rand.Intn(300) + 100
		_, err := c.page.Evaluate(fmt.Sprintf("window.scrollBy(0, %d)", scrollY))
		if err != nil {
			return err
		}
		time.Sleep(time.Duration(500+rand.Intn(500)) * time.Millisecond)
	}

	// 随机鼠标移动
	err := c.page.Mouse().Move(float64(rand.Intn(500)+100), float64(rand.Intn(300)+100))
	if err != nil {
		return err
	}

	return nil
}

// SafeGoto 安全导航（带反爬检测）
func (c *PooledContext) SafeGoto(url string, options ...playwright.PageGotoOptions) error {
	if c.page == nil {
		return fmt.Errorf("page not created")
	}

	// 模拟人类行为前等待
	c.HumanLikeDelay(500 * time.Millisecond)

	_, err := c.page.Goto(url, options...)
	if err != nil {
		return err
	}

	// 页面加载后模拟人类行为
	if err := c.SimulateHumanBehavior(); err != nil {
		utils.Warn(fmt.Sprintf("[-] 模拟人类行为失败: %v", err))
	}

	// 检测验证码
	if detected, captchaType, _ := c.DetectCaptcha(); detected {
		return fmt.Errorf("检测到%s，需要人工处理", captchaType)
	}

	// 检测反爬
	if detected, message, _ := c.DetectAntiBot(); detected {
		return fmt.Errorf("检测到反爬: %s", message)
	}

	return nil
}

// SafeClick 安全点击（带随机延迟）
func (c *PooledContext) SafeClick(selector string) error {
	if c.page == nil {
		return fmt.Errorf("page not created")
	}

	c.HumanLikeDelay(300 * time.Millisecond)

	if err := c.page.Locator(selector).Click(); err != nil {
		return err
	}

	c.HumanLikeDelay(200 * time.Millisecond)
	return nil
}

// SafeFill 安全填写（模拟人类输入）
func (c *PooledContext) SafeFill(selector, text string) error {
	if c.page == nil {
		return fmt.Errorf("page not created")
	}

	// 先点击输入框
	if err := c.SafeClick(selector); err != nil {
		return err
	}

	// 清空内容
	if err := c.page.Keyboard().Press("Control+KeyA"); err != nil {
		return err
	}
	if err := c.page.Keyboard().Press("Delete"); err != nil {
		return err
	}

	// 模拟人类输入
	return c.HumanLikeTyping(text)
}

// findLocalChrome 查找本地 Chrome
func findLocalChrome() string {
	paths := []string{
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		os.Getenv("LOCALAPPDATA") + `\Google\Chrome\Application\chrome.exe`,
		os.Getenv("PROGRAMFILES") + `\Google\Chrome\Application\chrome.exe`,
		os.Getenv("PROGRAMFILES(X86)") + `\Google\Chrome\Application\chrome.exe`,
	}

	for _, path := range paths {
		if path != "" {
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}
	return ""
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
