package utils

import (
	"Fuploader/internal/config"
	"Fuploader/internal/types"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
)

// LogServiceInterface 日志服务接口（避免循环依赖）
type LogServiceInterface interface {
	Add(log types.SimpleLog)
}

type Logger struct {
	file       *os.File
	logService LogServiceInterface
	mutex      sync.Mutex
}

var defaultLogger *Logger

func InitLogger() error {
	logPath := filepath.Join(config.Config.LogPath, fmt.Sprintf("app_%s.log", time.Now().Format("20060102")))
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defaultLogger = &Logger{file: file}
	return nil
}

func GetLogger() *Logger {
	if defaultLogger == nil {
		_ = InitLogger()
	}
	return defaultLogger
}

// SetLogService 设置日志服务，用于前端日志输出
func SetLogService(service LogServiceInterface) {
	GetLogger().mutex.Lock()
	defer GetLogger().mutex.Unlock()
	GetLogger().logService = service
}

// log 内部日志记录方法
func (l *Logger) log(level types.LogLevel, platform, msg string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	
	// 构建文件日志行
	var line string
	if platform != "" {
		line = fmt.Sprintf("[%s] [%s] [%s] %s\n", timestamp, level, platform, msg)
	} else {
		line = fmt.Sprintf("[%s] [%s] %s\n", timestamp, level, msg)
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	// 写入文件
	_, _ = l.file.WriteString(line)

	// 同时输出到前端
	if l.logService != nil {
		l.logService.Add(types.SimpleLog{
			Date:     time.Now().Format("2006/1/2"),
			Time:     time.Now().Format("15:04:05"),
			Message:  msg,
			Platform: platform,
			Level:    level,
		})
	}
}

// ========== 基础日志函数（不带平台）==========

func (l *Logger) Info(msg string) {
	l.log(types.LogLevelInfo, "", msg)
}

func (l *Logger) Error(msg string) {
	l.log(types.LogLevelError, "", msg)
}

func (l *Logger) Warn(msg string) {
	l.log(types.LogLevelWarn, "", msg)
}

func (l *Logger) Debug(msg string) {
	l.log(types.LogLevelDebug, "", msg)
}

func (l *Logger) Success(msg string) {
	l.log(types.LogLevelSuccess, "", msg)
}

// ========== 带平台的日志函数 ==========

func (l *Logger) InfoWithPlatform(platform, msg string) {
	l.log(types.LogLevelInfo, platform, msg)
}

func (l *Logger) ErrorWithPlatform(platform, msg string) {
	l.log(types.LogLevelError, platform, msg)
}

func (l *Logger) WarnWithPlatform(platform, msg string) {
	l.log(types.LogLevelWarn, platform, msg)
}

func (l *Logger) DebugWithPlatform(platform, msg string) {
	l.log(types.LogLevelDebug, platform, msg)
}

func (l *Logger) SuccessWithPlatform(platform, msg string) {
	l.log(types.LogLevelSuccess, platform, msg)
}

// ========== 全局便捷函数（不带平台）==========

func Info(msg string) {
	GetLogger().Info(msg)
}

func Error(msg string) {
	GetLogger().Error(msg)
}

func Warn(msg string) {
	GetLogger().Warn(msg)
}

func Debug(msg string) {
	GetLogger().Debug(msg)
}

func Success(msg string) {
	GetLogger().Success(msg)
}

// ========== 全局便捷函数（带平台）==========

func InfoWithPlatform(platform, msg string) {
	GetLogger().InfoWithPlatform(platform, msg)
}

func ErrorWithPlatform(platform, msg string) {
	GetLogger().ErrorWithPlatform(platform, msg)
}

func WarnWithPlatform(platform, msg string) {
	GetLogger().WarnWithPlatform(platform, msg)
}

func DebugWithPlatform(platform, msg string) {
	GetLogger().DebugWithPlatform(platform, msg)
}

func SuccessWithPlatform(platform, msg string) {
	GetLogger().SuccessWithPlatform(platform, msg)
}

// Screenshot 截图并保存到日志目录
func Screenshot(page playwright.Page, name string) error {
	screenshotPath := filepath.Join(config.Config.LogPath, fmt.Sprintf("screenshot_%s_%s.png", time.Now().Format("20060102_150405"), name))
	_, err := page.Screenshot(playwright.PageScreenshotOptions{
		Path:     playwright.String(screenshotPath),
		FullPage: playwright.Bool(true),
	})
	if err != nil {
		Error(fmt.Sprintf("截图失败: %v", err))
		return err
	}
	Info(fmt.Sprintf("截图已保存: %s", screenshotPath))
	return nil
}
