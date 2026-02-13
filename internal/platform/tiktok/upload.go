package tiktok

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"Fuploader/internal/platform/browser"
	"Fuploader/internal/utils"

	"github.com/playwright-community/playwright-go"
)

func (u *Uploader) uploadVideo(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext, locatorBase playwright.Locator, videoPath string) error {
	utils.InfoWithPlatform(u.platform, "正在上传视频...")

	if _, err := os.Stat(videoPath); err != nil {
		return fmt.Errorf("视频文件不存在: %w", err)
	}

	locators := GetLocators()
	uploadButton := u.findFirstVisibleLocator(locatorBase, locators.UploadButton)
	if uploadButton == nil {
		return fmt.Errorf("未找到上传按钮")
	}

	if err := uploadButton.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(10000),
	}); err != nil {
		return fmt.Errorf("等待上传按钮失败: %w", err)
	}

	fileChooser, err := page.ExpectFileChooser(func() error {
		return uploadButton.Click()
	})
	if err != nil {
		return fmt.Errorf("等待文件选择器失败: %w", err)
	}

	if err := fileChooser.SetFiles(videoPath); err != nil {
		return fmt.Errorf("设置视频文件失败: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "等待视频上传完成...")
	if err := u.waitForUploadComplete(ctx, page, browserCtx, locatorBase); err != nil {
		return err
	}

	utils.InfoWithPlatform(u.platform, "视频上传完成")
	return nil
}

func (u *Uploader) waitForUploadComplete(ctx context.Context, page playwright.Page, browserCtx *browser.PooledContext, locatorBase playwright.Locator) error {
	uploadTimeout := 5 * time.Minute
	uploadCheckInterval := 2 * time.Second
	uploadStartTime := time.Now()

	for time.Since(uploadStartTime) < uploadTimeout {
		select {
		case <-ctx.Done():
			return fmt.Errorf("上传已取消")
		default:
		}

		if browserCtx.IsPageClosed() {
			return fmt.Errorf("浏览器已关闭")
		}

		locators := GetLocators()
		postButton := u.findFirstVisibleLocator(locatorBase, locators.PostButton)
		if postButton != nil {
			disabledAttr, err := postButton.GetAttribute("disabled")
			if err == nil && (disabledAttr == "" || disabledAttr == "false") {
				utils.InfoWithPlatform(u.platform, "发布按钮已可用，上传完成")
				return nil
			}
		}

		videoPreview := locatorBase.Locator("video, .video-preview").First()
		if count, _ := videoPreview.Count(); count > 0 {
			if visible, _ := videoPreview.IsVisible(); visible {
				return nil
			}
		}

		selectFileBtn := locatorBase.Locator("button[aria-label='Select file']")
		if count, _ := selectFileBtn.Count(); count > 0 {
			utils.WarnWithPlatform(u.platform, "检测到上传错误，可能需要重试")
		}

		time.Sleep(uploadCheckInterval)
	}

	return fmt.Errorf("上传超时")
}

func (u *Uploader) fillTitleAndDescription(locatorBase playwright.Locator, title, description string) error {
	utils.InfoWithPlatform(u.platform, "填写标题...")

	locators := GetLocators()
	editorLocator := u.findFirstVisibleLocator(locatorBase, locators.Editor)
	if editorLocator == nil {
		return fmt.Errorf("未找到编辑器")
	}

	if err := editorLocator.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("未找到编辑器: %w", err)
	}

	if err := editorLocator.Click(); err != nil {
		return fmt.Errorf("点击编辑器失败: %w", err)
	}
	time.Sleep(500 * time.Millisecond)

	page, err := editorLocator.Page()
	if err != nil {
		return fmt.Errorf("获取页面失败: %w", err)
	}

	page.Keyboard().Press("End")
	page.Keyboard().Press("Control+KeyA")
	page.Keyboard().Press("Delete")
	page.Keyboard().Press("End")
	time.Sleep(500 * time.Millisecond)

	content := title
	if description != "" {
		content += "\n\n" + description
	}

	page.Keyboard().Type(content)
	time.Sleep(500 * time.Millisecond)
	page.Keyboard().Press("End")
	page.Keyboard().Press("Enter")

	utils.InfoWithPlatform(u.platform, fmt.Sprintf("标题已填写: %s", title))

	if description != "" {
		utils.InfoWithPlatform(u.platform, "描述已填写")
	}

	return nil
}

func (u *Uploader) addTags(locatorBase playwright.Locator, tags []string) error {
	utils.InfoWithPlatform(u.platform, "添加标签...")

	page, err := locatorBase.Page()
	if err != nil {
		return fmt.Errorf("获取页面失败: %w", err)
	}

	for _, tag := range tags {
		cleanTag := strings.TrimSpace(tag)
		cleanTag = strings.ReplaceAll(cleanTag, "#", "")
		if cleanTag == "" {
			continue
		}

		page.Keyboard().Press("End")
		time.Sleep(500 * time.Millisecond)
		page.Keyboard().Type("#" + cleanTag + " ")
		page.Keyboard().Press("Space")
		time.Sleep(500 * time.Millisecond)
		page.Keyboard().Press("Backspace")
		page.Keyboard().Press("End")
	}

	utils.InfoWithPlatform(u.platform, "标签添加完成")
	return nil
}

func (u *Uploader) setCover(page playwright.Page, locatorBase playwright.Locator, coverPath string) error {
	if _, err := os.Stat(coverPath); err != nil {
		return fmt.Errorf("封面文件不存在: %w", err)
	}

	utils.InfoWithPlatform(u.platform, "设置封面...")

	locators := GetLocators()
	coverContainer := u.findFirstVisibleLocator(locatorBase, locators.CoverContainer)
	if coverContainer == nil {
		return fmt.Errorf("未找到封面区域")
	}

	if err := coverContainer.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		return fmt.Errorf("未找到封面区域: %w", err)
	}

	if err := coverContainer.Click(); err != nil {
		return fmt.Errorf("点击封面区域失败: %w", err)
	}
	time.Sleep(2 * time.Second)

	uploadCoverBtn := u.findFirstVisibleLocator(locatorBase, locators.UploadCover)
	if uploadCoverBtn != nil {
		uploadCoverBtn.Click()
		time.Sleep(1 * time.Second)
	}

	fileChooser, err := page.ExpectFileChooser(func() error {
		uploadBtn := locatorBase.Locator("button:has-text('Upload'):visible").First()
		return uploadBtn.Click()
	})
	if err != nil {
		return fmt.Errorf("等待文件选择器失败: %w", err)
	}

	if err := fileChooser.SetFiles(coverPath); err != nil {
		return fmt.Errorf("上传封面失败: %w", err)
	}

	time.Sleep(3 * time.Second)

	confirmBtn := u.findFirstVisibleLocator(locatorBase, locators.ConfirmCover)
	if confirmBtn != nil {
		confirmBtn.Click()
		time.Sleep(1 * time.Second)
	}

	utils.InfoWithPlatform(u.platform, "封面设置完成")
	return nil
}

func (u *Uploader) findFirstVisibleLocator(base playwright.Locator, selectors []string) playwright.Locator {
	for _, selector := range selectors {
		locator := base.Locator(selector)
		if count, _ := locator.Count(); count > 0 {
			if visible, _ := locator.First().IsVisible(); visible {
				return locator.First()
			}
		}
	}
	return nil
}
