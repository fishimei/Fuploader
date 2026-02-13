package tencent

import (
	"fmt"
	"time"

	"github.com/playwright-community/playwright-go"
)

func (u *Uploader) addToCollection(page playwright.Page, collection string) error {
	collectionSection := page.Locator("div.form-item:has(div.form-label:has-text('添加到合集'))")

	if count, _ := collectionSection.Count(); count == 0 {
		return nil
	}

	collectionContent := collectionSection.Locator("div.form-content").First()

	collectionElements := collectionContent.Locator(".option-list-wrap > div")
	count, err := collectionElements.Count()
	if err != nil {
		return fmt.Errorf("失败: 添加到合集 - 获取合集列表失败: %w", err)
	}

	if count > 1 {
		if err := collectionContent.Click(); err != nil {
			return fmt.Errorf("失败: 添加到合集 - 点击合集选项失败: %w", err)
		}
		time.Sleep(1 * time.Second)

		if err := collectionElements.First().Click(); err != nil {
			return fmt.Errorf("失败: 添加到合集 - 选择合集失败: %w", err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	return nil
}
