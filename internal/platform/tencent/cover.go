package tencent

import (
	"fmt"
	"os"
	"strings"
	"time"

	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

func (u *Uploader) setCover(page playwright.Page, coverPath string) error {
	if _, err := os.Stat(coverPath); err != nil {
		return fmt.Errorf("失败: 设置封面 - 封面文件不存在: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "设置封面...")

	coverBtn := page.GetByText("设置封面").First()
	if err := coverBtn.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("失败: 设置封面 - 未找到封面设置按钮: %w", err)
	}

	if err := coverBtn.Click(); err != nil {
		return fmt.Errorf("失败: 设置封面 - 点击封面设置按钮失败: %w", err)
	}
	time.Sleep(2 * time.Second)

	coverInput := page.Locator("input[type='file'][accept*='image']").First()
	if err := coverInput.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(u.config.ElementWaitTimeout.Milliseconds())),
	}); err != nil {
		return fmt.Errorf("失败: 设置封面 - 未找到封面文件输入框: %w", err)
	}

	if err := coverInput.SetInputFiles(coverPath); err != nil {
		return fmt.Errorf("失败: 设置封面 - 上传封面失败: %w", err)
	}

	time.Sleep(3 * time.Second)

	finishBtn := page.Locator("button:has-text('完成'), button:has-text('确定')").First()
	if count, _ := finishBtn.Count(); count > 0 {
		if err := finishBtn.Click(); err != nil {
			utils.WarnWithPlatform(u.platform, fmt.Sprintf("失败: 设置封面 - 点击完成按钮失败: %v", err))
		}
	}
	time.Sleep(1 * time.Second)

	utils.InfoWithPlatform(u.platform, "封面设置完成")
	return nil
}

func formatStrForShortTitle(originTitle string, minLength, maxLength int) string {
	allowedSpecialChars := "《》" + "：+?%°"

	var filteredChars []rune
	for _, char := range originTitle {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
			filteredChars = append(filteredChars, char)
		} else if char >= 0x4e00 && char <= 0x9fff {
			filteredChars = append(filteredChars, char)
		} else if char == ',' {
			filteredChars = append(filteredChars, ' ')
		} else {
			allowed := false
			for _, allowedChar := range allowedSpecialChars {
				if char == allowedChar {
					allowed = true
					break
				}
			}
			if allowed {
				filteredChars = append(filteredChars, char)
			}
		}
	}
	formattedString := string(filteredChars)

	runeLen := len([]rune(formattedString))
	if runeLen > maxLength {
		runes := []rune(formattedString)
		formattedString = string(runes[:maxLength])
	} else if runeLen < minLength {
		spacesNeeded := minLength - runeLen
		formattedString += strings.Repeat(" ", spacesNeeded)
	}

	return formattedString
}

func (u *Uploader) setShortTitle(page playwright.Page, title string) error {
	shortTitle := formatStrForShortTitle(title, u.config.ShortTitleMinLength, u.config.ShortTitleMaxLength)

	shortTitleInput := page.Locator("input[placeholder*='字数建议6-16个字符']").First()

	if count, _ := shortTitleInput.Count(); count > 0 {
		if err := shortTitleInput.Fill(shortTitle); err != nil {
			return fmt.Errorf("失败: 设置短标题 - 填写短标题失败: %w", err)
		}
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}
