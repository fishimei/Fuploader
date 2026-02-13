package bilibili

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"Fuploader/internal/utils"

	"github.com/imroc/req/v3"
	"github.com/tidwall/gjson"
)

type PlaywrightCookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires"`
	HttpOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
	SameSite string  `json:"sameSite"`
}

type PlaywrightStorageState struct {
	Cookies []PlaywrightCookie `json:"cookies"`
	Origins []struct {
		Origin       string `json:"origin"`
		LocalStorage []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"localStorage"`
	} `json:"origins"`
}

func convertPlaywrightCookies(cookies []PlaywrightCookie) string {
	var parts []string
	for _, c := range cookies {
		parts = append(parts, fmt.Sprintf("%s=%s", c.Name, c.Value))
	}
	return strings.Join(parts, "; ")
}

func ValidateCookieAPI(cookiePath string) (bool, string, error) {
	utils.InfoWithPlatform("bilibili", fmt.Sprintf("验证Cookie(API) - 开始验证，cookie路径: %s", cookiePath))

	loginInfo, err := os.ReadFile(cookiePath)
	if err != nil || len(loginInfo) == 0 {
		utils.WarnWithPlatform("bilibili", fmt.Sprintf("验证Cookie(API) - 读取cookie文件失败: %v", err))
		return false, "", fmt.Errorf("cookie文件不存在")
	}

	utils.InfoWithPlatform("bilibili", fmt.Sprintf("验证Cookie(API) - 读取到cookie文件，大小: %d 字节", len(loginInfo)))

	var storageState PlaywrightStorageState
	if err := json.Unmarshal(loginInfo, &storageState); err != nil {
		utils.WarnWithPlatform("bilibili", fmt.Sprintf("验证Cookie(API) - 解析cookie文件失败: %v", err))
		return false, "", fmt.Errorf("解析cookie文件失败: %w", err)
	}

	if len(storageState.Cookies) == 0 {
		utils.WarnWithPlatform("bilibili", "验证Cookie(API) - cookie文件中没有cookies")
		return false, "", fmt.Errorf("cookie文件中没有cookies")
	}

	utils.InfoWithPlatform("bilibili", fmt.Sprintf("验证Cookie(API) - 共 %d 个cookie", len(storageState.Cookies)))
	cookie := convertPlaywrightCookies(storageState.Cookies)

	return validateCookieString(cookie)
}

func validateCookieString(cookie string) (bool, string, error) {
	utils.InfoWithPlatform("bilibili", "验证Cookie(API) - 开始请求API验证cookie")

	client := req.C().SetCommonHeaders(map[string]string{
		"user-agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"cookie":     cookie,
		"Referer":    "https://www.bilibili.com",
	})

	resp, err := client.R().Get("https://api.bilibili.com/x/web-interface/nav")
	if err != nil {
		utils.WarnWithPlatform("bilibili", fmt.Sprintf("验证Cookie(API) - API请求失败: %v", err))
		return false, "", err
	}

	respBody := resp.Bytes()
	utils.InfoWithPlatform("bilibili", fmt.Sprintf("验证Cookie(API) - API响应: %s", string(respBody)))

	code := gjson.GetBytes(respBody, "code").Int()
	message := gjson.GetBytes(respBody, "message").String()

	if code != 0 {
		utils.WarnWithPlatform("bilibili", fmt.Sprintf("验证Cookie(API) - API返回错误: code=%d, message=%s", code, message))
		return false, "", fmt.Errorf("API返回错误: code=%d, message=%s", code, message)
	}

	isLogin := gjson.GetBytes(respBody, "data.isLogin").Bool()
	uname := gjson.GetBytes(respBody, "data.uname").String()

	utils.InfoWithPlatform("bilibili", fmt.Sprintf("验证Cookie(API) - 验证结果: isLogin=%v, uname=%s", isLogin, uname))

	return isLogin, uname, nil
}
