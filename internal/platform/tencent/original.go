package tencent

import (
	"fmt"
	"time"

	"github.com/playwright-community/playwright-go"
)

func (u *Uploader) setOriginal(page playwright.Page, category string) error {
	originalCheckbox := page.GetByLabel("视频为原创").First()
	if count, _ := originalCheckbox.Count(); count > 0 {
		if err := originalCheckbox.Check(); err != nil {
			return fmt.Errorf("失败: 设置原创声明 - 勾选原创声明失败: %w", err)
		}
	}

	agreementLabel := page.Locator("label:has-text('我已阅读并同意 《视频号原创声明使用条款》')").First()
	isVisible, _ := agreementLabel.IsVisible()
	if isVisible {
		agreementCheckbox := page.GetByLabel("我已阅读并同意 《视频号原创声明使用条款》").First()
		if err := agreementCheckbox.Check(); err != nil {
			return fmt.Errorf("失败: 设置原创声明 - 勾选使用条款失败: %w", err)
		}

		declareBtn := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "声明原创"}).First()
		if err := declareBtn.Click(); err != nil {
			return fmt.Errorf("失败: 设置原创声明 - 点击声明原创按钮失败: %w", err)
		}
	}

	newOriginalLabel := page.Locator("div.label span:has-text('声明原创')").First()
	if count, _ := newOriginalLabel.Count(); count > 0 && category != "" {
		originalCheckboxNew := page.Locator("div.declare-original-checkbox input.ant-checkbox-input").First()
		isDisabled, _ := originalCheckboxNew.IsDisabled()

		if !isDisabled {
			if err := originalCheckboxNew.Click(); err != nil {
				return fmt.Errorf("失败: 设置原创声明 - 点击新版原创复选框失败: %w", err)
			}

			checkedWrapper := page.Locator("div.declare-original-dialog label.ant-checkbox-wrapper.ant-checkbox-wrapper-checked:visible").First()
			if count, _ := checkedWrapper.Count(); count == 0 {
				visibleCheckbox := page.Locator("div.declare-original-dialog input.ant-checkbox-input:visible").First()
				visibleCheckbox.Click()
			}
		}

		originalTypeForm := page.Locator("div.original-type-form > div.form-label:has-text('原创类型'):visible").First()
		if count, _ := originalTypeForm.Count(); count > 0 {
			dropdown := page.Locator("div.form-content:visible").First()
			if err := dropdown.Click(); err != nil {
				return fmt.Errorf("失败: 设置原创声明 - 点击原创类型下拉菜单失败: %w", err)
			}
			time.Sleep(1 * time.Second)

			typeOption := page.Locator(fmt.Sprintf("div.form-content:visible ul.weui-desktop-dropdown__list li.weui-desktop-dropdown__list-ele:has-text('%s')", category)).First()
			if err := typeOption.Click(); err != nil {
				return fmt.Errorf("失败: 设置原创声明 - 选择原创类型失败: %w", err)
			}
			time.Sleep(1 * time.Second)
		}

		declareBtnNew := page.Locator("button:has-text('声明原创'):visible").First()
		if count, _ := declareBtnNew.Count(); count > 0 {
			if err := declareBtnNew.Click(); err != nil {
				return fmt.Errorf("失败: 设置原创声明 - 点击新版声明原创按钮失败: %w", err)
			}
		}
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}
