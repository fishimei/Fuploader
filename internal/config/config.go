package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type AppConfig struct {
	DbPath            string
	CookiePath        string
	VideoPath         string
	LogPath           string
	ThumbnailPath     string
	UploadConcurrency int
	DefaultTimeout    int
	DebugMode         bool // 调试模式开关
	Headless          bool // 浏览器无头模式开关（true=隐藏浏览器窗口）
}

var Config *AppConfig

func Init() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	baseDir := filepath.Dir(exePath)

	// 定义存储目录（数据库文件所在的目录）
	storageDir := filepath.Join(baseDir, "storage")

	Config = &AppConfig{
		DbPath:            filepath.Join(baseDir, DefaultDbPath),
		CookiePath:        filepath.Join(baseDir, DefaultCookiePath),
		VideoPath:         filepath.Join(baseDir, DefaultVideoPath),
		LogPath:           filepath.Join(baseDir, DefaultLogPath),
		ThumbnailPath:     filepath.Join(baseDir, DefaultThumbnailPath),
		UploadConcurrency: UploadConcurrency,
		DefaultTimeout:    DefaultTimeout,
		DebugMode:         os.Getenv("FUPLOADER_DEBUG") == "true", // 通过环境变量控制调试模式
		Headless:          os.Getenv("FUPLOADER_HEADLESS") == "true", // 通过环境变量控制无头模式
	}

	// 创建目录（只创建目录，不包括数据库文件路径）
	dirs := []string{
		storageDir,           // 存储根目录
		Config.CookiePath,    // cookies 目录
		Config.VideoPath,     // videos 目录
		Config.LogPath,       // logs 目录
		Config.ThumbnailPath, // thumbnails 目录
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s failed: %w", dir, err)
		}
	}

	return nil
}

func GetDbPath() string {
	return Config.DbPath
}

func GetCookiePath(platform string, accountID int) string {
	return filepath.Join(Config.CookiePath, fmt.Sprintf("%s_%d.json", platform, accountID))
}
