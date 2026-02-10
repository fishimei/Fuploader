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

// PlaywrightCookie Playwright的cookie格式
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

// PlaywrightStorageState Playwright StorageState格式
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

// convertPlaywrightCookies 转换Playwright cookie格式为字符串
func convertPlaywrightCookies(cookies []PlaywrightCookie) string {
	var parts []string
	for _, c := range cookies {
		parts = append(parts, fmt.Sprintf("%s=%s", c.Name, c.Value))
	}
	return strings.Join(parts, "; ")
}

// ValidateCookie 验证Cookie是否有效
func ValidateCookie(cookiePath string) (bool, string, error) {
	utils.Info(fmt.Sprintf("[-] B站验证 - 开始验证，cookie路径: %s", cookiePath))

	loginInfo, err := os.ReadFile(cookiePath)
	if err != nil || len(loginInfo) == 0 {
		utils.Error(fmt.Sprintf("[-] B站验证 - 读取cookie文件失败: %v", err))
		return false, "", fmt.Errorf("cookie文件不存在")
	}

	utils.Info(fmt.Sprintf("[-] B站验证 - 读取到cookie文件，大小: %d 字节", len(loginInfo)))

	// 解析Playwright StorageState格式
	var storageState PlaywrightStorageState
	if err := json.Unmarshal(loginInfo, &storageState); err != nil {
		utils.Error(fmt.Sprintf("[-] B站验证 - 解析cookie文件失败: %v", err))
		return false, "", fmt.Errorf("解析cookie文件失败: %w", err)
	}

	if len(storageState.Cookies) == 0 {
		utils.Error("[-] B站验证 - cookie文件中没有cookies")
		return false, "", fmt.Errorf("cookie文件中没有cookies")
	}

	utils.Info(fmt.Sprintf("[-] B站验证 - 共 %d 个cookie", len(storageState.Cookies)))
	cookie := convertPlaywrightCookies(storageState.Cookies)
	utils.Info(fmt.Sprintf("[-] B站验证 - 转换后的cookie字符串长度: %d", len(cookie)))

	return validateCookieString(cookie)
}

// validateCookieString 验证cookie字符串
func validateCookieString(cookie string) (bool, string, error) {
	utils.Info("[-] B站验证 - 开始请求API验证cookie")

	client := req.C().SetCommonHeaders(map[string]string{
		"user-agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"cookie":     cookie,
		"Referer":    "https://www.bilibili.com",
	})

	resp, err := client.R().Get("https://api.bilibili.com/x/web-interface/nav")
	if err != nil {
		utils.Error(fmt.Sprintf("[-] B站验证 - API请求失败: %v", err))
		return false, "", err
	}

	respBody := resp.Bytes()
	utils.Info(fmt.Sprintf("[-] B站验证 - API响应: %s", string(respBody)))

	isLogin := gjson.GetBytes(respBody, "data.isLogin").Bool()
	uname := gjson.GetBytes(respBody, "data.uname").String()
	code := gjson.GetBytes(respBody, "code").Int()
	message := gjson.GetBytes(respBody, "message").String()

	utils.Info(fmt.Sprintf("[-] B站验证 - 解析结果: code=%d, message=%s, isLogin=%v, uname=%s", code, message, isLogin, uname))

	return isLogin, uname, nil
}
