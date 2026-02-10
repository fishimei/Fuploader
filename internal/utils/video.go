package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// CheckFFmpeg 检查系统是否安装了 ffmpeg
func CheckFFmpeg() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

// ExtractFirstFrame 从视频抽取第一帧作为封面
// 使用系统 ffmpeg 命令
// 返回生成的封面路径，如果失败返回错误
func ExtractFirstFrame(videoPath string) (string, error) {
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		return "", fmt.Errorf("视频文件不存在: %s", videoPath)
	}

	// 检查 ffmpeg 是否可用
	if !CheckFFmpeg() {
		return "", fmt.Errorf("系统未安装 ffmpeg，无法抽取封面")
	}

	tempDir := os.TempDir()
	coverFileName := fmt.Sprintf("video_cover_%d_%d.jpg", time.Now().Unix(), time.Now().Nanosecond())
	coverPath := filepath.Join(tempDir, coverFileName)

	// 使用 ffmpeg 命令抽取第一帧
	// -ss 00:00:01 从第1秒开始（避免黑帧）
	// -vframes 1 只抽取一帧
	// -q:v 2 设置图片质量（2是高质量）
	// -y 覆盖已存在文件
	// 注意：-ss 放在 -i 之前是快速定位，放在之后是精确但慢速
	cmd := exec.Command("ffmpeg", "-ss", "00:00:01", "-i", videoPath, "-vframes", "1", "-q:v", "2", "-y", coverPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg 执行失败: %v, 输出: %s", err, string(output))
	}

	// 检查封面文件是否生成成功
	if _, err := os.Stat(coverPath); os.IsNotExist(err) {
		return "", fmt.Errorf("封面文件生成失败")
	}

	// 检查文件大小，确保不是空文件
	fileInfo, err := os.Stat(coverPath)
	if err != nil || fileInfo.Size() == 0 {
		return "", fmt.Errorf("封面文件生成失败或为空")
	}

	return coverPath, nil
}

// ExtractFrameAt 从视频指定时间点抽取帧作为封面
// timeSeconds: 时间点（秒）
func ExtractFrameAt(videoPath string, timeSeconds int) (string, error) {
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		return "", fmt.Errorf("视频文件不存在: %s", videoPath)
	}

	// 检查 ffmpeg 是否可用
	if !CheckFFmpeg() {
		return "", fmt.Errorf("系统未安装 ffmpeg，无法抽取封面")
	}

	tempDir := os.TempDir()
	coverFileName := fmt.Sprintf("video_cover_%d_%d.jpg", time.Now().Unix(), time.Now().Nanosecond())
	coverPath := filepath.Join(tempDir, coverFileName)

	// 格式化时间字符串（HH:MM:SS）
	timeStr := fmt.Sprintf("%02d:%02d:%02d", timeSeconds/3600, (timeSeconds%3600)/60, timeSeconds%60)

	Info(fmt.Sprintf("[抽帧] 视频: %s, 时间点: %s, 输出: %s", videoPath, timeStr, coverPath))

	// 执行 ffmpeg 命令抽取指定时间帧
	// -ss 放在 -i 之前是快速定位，适合抽帧
	// -accurate_seek 配合 -ss 在 -i 之前使用，提高精度
	cmd := exec.Command("ffmpeg", "-ss", timeStr, "-i", videoPath, "-vframes", "1", "-q:v", "2", "-y", coverPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		Error(fmt.Sprintf("[抽帧失败] ffmpeg 错误: %v, 输出: %s", err, string(output)))
		return "", fmt.Errorf("ffmpeg 执行失败: %v, 输出: %s", err, string(output))
	}

	// 检查封面文件是否生成成功
	if _, err := os.Stat(coverPath); os.IsNotExist(err) {
		Error(fmt.Sprintf("[抽帧失败] 封面文件未生成: %s", coverPath))
		return "", fmt.Errorf("封面文件生成失败")
	}

	// 检查文件大小
	fileInfo, err := os.Stat(coverPath)
	if err != nil || fileInfo.Size() == 0 {
		Error(fmt.Sprintf("[抽帧失败] 封面文件为空: %s", coverPath))
		return "", fmt.Errorf("封面文件生成失败或为空")
	}

	Info(fmt.Sprintf("[抽帧成功] 封面已生成: %s, 大小: %d bytes", coverPath, fileInfo.Size()))

	return coverPath, nil
}

// CleanupTempFile 清理临时文件
func CleanupTempFile(filePath string) {
	if filePath != "" {
		os.Remove(filePath)
	}
}
