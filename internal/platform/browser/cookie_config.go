package browser

// CookieDomainConfig 单个域名的Cookie配置
type CookieDomainConfig struct {
	Domain          string   // Cookie域名
	RequiredCookies []string // 必需Cookie名称列表
	ExtendedCookies []string // 扩展Cookie名称列表
}

// PlatformCookieConfig 平台Cookie检测配置
type PlatformCookieConfig struct {
	Domains         []CookieDomainConfig // 多域名Cookie配置
	RequiredCookies []string             // 必需Cookie名称列表（兼容旧版单域名）
	ExtendedCookies []string             // 扩展Cookie名称列表（兼容旧版单域名）
	ValidateURL     string               // 验证URL（从cookie_config读取）
}

// PlatformCookieConfigs 各平台的Cookie配置
// 包含必需Cookie（维持登录态）和扩展Cookie（操作/风控）
// ValidateURL 待配置，目前均为空
var PlatformCookieConfigs = map[string]PlatformCookieConfig{
	"bilibili": {
		RequiredCookies: []string{"SESSDATA"},
		ExtendedCookies: []string{"bili_jct", "DedeUserID"},
		ValidateURL:     "",
	},
	"douyin": {
		RequiredCookies: []string{"sessionid"},
		ExtendedCookies: []string{"ttwid", "odin_tt"},
		ValidateURL:     "",
	},
	"tiktok": {
		RequiredCookies: []string{"sessionid"},
		ExtendedCookies: []string{"_ttp", "tt_chain_token"},
		ValidateURL:     "",
	},
	"kuaishou": {
		RequiredCookies: []string{"kuaishou.web.cp.api_ph", "kuaishou.web.cp.api_st"},
		ExtendedCookies: []string{"did"},
		ValidateURL:     "",
	},
	"tencent": {
		// 微信视频号Cookie配置
		// 基于实际测试，视频号核心Cookie为：
		// channels.weixin.qq.com: sessionid + wxuin（实际获取到的核心字段）
		Domains: []CookieDomainConfig{
			{
				Domain:          "https://channels.weixin.qq.com",
				RequiredCookies: []string{"sessionid", "wxuin"},
				ExtendedCookies: []string{},
			},
		},
		ValidateURL: "",
	},
	"baijiahao": {
		RequiredCookies: []string{"PTOKEN"},
		ExtendedCookies: []string{"BAIDUID", "STOKEN", "BDUSS"},
		ValidateURL:     "",
	},
	"xiaohongshu": {
		// 小红书Cookie配置
		// 一、核心必须Cookie（身份与会话，缺失则登录态失效）
		// .xiaohongshu.com: web_session（全平台登录态核心）、a1（设备标识）、customer-sso-sid（单点登录令牌）
		// creator.xiaohongshu.com: galaxy_creator_session_id（创作者会话）、galaxy.creator.beaker.session.id（创作者校验）、
		//                          x-user-id-creator.xiaohongshu.com（创作者账号ID）、access-token-creator.xiaohongshu.com（API访问令牌）
		// 二、辅助Cookie（不影响登录，但影响功能/风控）
		// loadts（加载时间戳）、websectiga（安全校验）、webBuild（版本号）、webId（临时标识）
		Domains: []CookieDomainConfig{
			{
				Domain:          "https://xiaohongshu.com",
				RequiredCookies: []string{"web_session", "a1", "customer-sso-sid"},
				ExtendedCookies: []string{"loadts", "websectiga", "webBuild", "webId"},
			},
			{
				Domain:          "https://creator.xiaohongshu.com",
				RequiredCookies: []string{"galaxy_creator_session_id", "galaxy.creator.beaker.session.id", "x-user-id-creator.xiaohongshu.com", "access-token-creator.xiaohongshu.com"},
				ExtendedCookies: []string{},
			},
		},
		ValidateURL: "",
	},
}

// GetCookieConfig 获取指定平台的Cookie配置
func GetCookieConfig(platform string) (PlatformCookieConfig, bool) {
	config, ok := PlatformCookieConfigs[platform]
	return config, ok
}

// GetAllCookies 获取所有需要保存的Cookie（必需+扩展）
func (config PlatformCookieConfig) GetAllCookies() []string {
	// 如果有多域名配置，合并所有域名的Cookie
	if len(config.Domains) > 0 {
		allCookies := make([]string, 0)
		for _, domain := range config.Domains {
			allCookies = append(allCookies, domain.RequiredCookies...)
			allCookies = append(allCookies, domain.ExtendedCookies...)
		}
		return allCookies
	}
	// 兼容旧版单域名配置
	allCookies := make([]string, 0, len(config.RequiredCookies)+len(config.ExtendedCookies))
	allCookies = append(allCookies, config.RequiredCookies...)
	allCookies = append(allCookies, config.ExtendedCookies...)
	return allCookies
}

// GetAllDomains 获取所有需要保存Cookie的域名
func (config PlatformCookieConfig) GetAllDomains() []string {
	if len(config.Domains) > 0 {
		domains := make([]string, 0, len(config.Domains))
		for _, d := range config.Domains {
			domains = append(domains, d.Domain)
		}
		return domains
	}
	// 兼容旧版，返回空切片
	return []string{}
}
