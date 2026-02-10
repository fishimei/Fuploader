package app

import (
	"Fuploader/internal/config"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// FileLoader 自定义静态文件加载器
type FileLoader struct {
	http.Handler
}

// NewFileLoader 创建文件加载器
func NewFileLoader() *FileLoader {
	return &FileLoader{}
}

// ServeHTTP 处理 HTTP 请求
func (h *FileLoader) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// 处理缩略图请求 /thumbnails/
	if strings.HasPrefix(path, "/thumbnails/") {
		h.serveThumbnail(w, r, path)
		return
	}

	// 处理视频文件请求 /videos/
	if strings.HasPrefix(path, "/videos/") {
		h.serveVideo(w, r, path)
		return
	}

	// 其他请求返回 404
	http.NotFound(w, r)
}

// serveThumbnail 服务缩略图文件
func (h *FileLoader) serveThumbnail(w http.ResponseWriter, r *http.Request, path string) {
	// 从路径中提取文件名
	filename := filepath.Base(path)
	if filename == "" || filename == "." || filename == "/" {
		http.NotFound(w, r)
		return
	}

	// 安全检查：防止目录遍历攻击
	if strings.Contains(filename, "..") || strings.Contains(filename, "~/") {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	// 构建完整路径
	filePath := filepath.Join(config.Config.ThumbnailPath, filename)

	// 检查文件是否存在
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}

	// 设置 Content-Type
	contentType := getContentType(filePath)
	w.Header().Set("Content-Type", contentType)

	// 读取并返回文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read file: %v", err), http.StatusInternalServerError)
		return
	}

	w.Write(data)
}

// serveVideo 服务视频文件
func (h *FileLoader) serveVideo(w http.ResponseWriter, r *http.Request, path string) {
	filename := filepath.Base(path)
	if filename == "" || filename == "." || filename == "/" {
		http.NotFound(w, r)
		return
	}

	if strings.Contains(filename, "..") || strings.Contains(filename, "~/") {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(config.Config.VideoPath, filename)

	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}

	contentType := getContentType(filePath)
	w.Header().Set("Content-Type", contentType)

	data, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read file: %v", err), http.StatusInternalServerError)
		return
	}

	w.Write(data)
}

// getContentType 根据文件扩展名获取 Content-Type
func getContentType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".mp4":
		return "video/mp4"
	case ".mov":
		return "video/quicktime"
	case ".avi":
		return "video/x-msvideo"
	default:
		return "application/octet-stream"
	}
}
