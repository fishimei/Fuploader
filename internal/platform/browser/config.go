package browser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// PoolConfig 浏览器池配置
type PoolConfig struct {
	MaxBrowsers           int  `json:"max_browsers"`             // 最大浏览器实例数
	MaxContextsPerBrowser int  `json:"max_contexts_per_browser"` // 每个浏览器的最大上下文数
	ContextIdleTimeout    int  `json:"context_idle_timeout"`     // 上下文空闲超时时间（秒）
	EnableHealthCheck     bool `json:"enable_health_check"`      // 是否启用健康检查
	HealthCheckInterval   int  `json:"health_check_interval"`    // 健康检查间隔（秒）
}

// DefaultPoolConfig 默认配置
var DefaultPoolConfig = PoolConfig{
	MaxBrowsers:           2,
	MaxContextsPerBrowser: 5,
	ContextIdleTimeout:    30,
	EnableHealthCheck:     true,
	HealthCheckInterval:   60,
}

var (
	poolConfig     *PoolConfig
	configOnce     sync.Once
	configPath     string
)

// LoadPoolConfig 加载浏览器池配置
func LoadPoolConfig() *PoolConfig {
	configOnce.Do(func() {
		poolConfig = &PoolConfig{}
		*poolConfig = DefaultPoolConfig

		// 尝试从配置文件加载
		if configPath == "" {
			configPath = getDefaultConfigPath()
		}

		if _, err := os.Stat(configPath); err == nil {
			data, err := os.ReadFile(configPath)
			if err == nil {
				if err := json.Unmarshal(data, poolConfig); err != nil {
					fmt.Printf("[-] 解析浏览器池配置失败，使用默认配置: %v\n", err)
					*poolConfig = DefaultPoolConfig
				}
			}
		} else {
			// 配置文件不存在，创建默认配置
			SavePoolConfig(poolConfig)
		}
	})

	return poolConfig
}

// SavePoolConfig 保存浏览器池配置
func SavePoolConfig(cfg *PoolConfig) error {
	if configPath == "" {
		configPath = getDefaultConfigPath()
	}

	// 确保目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	poolConfig = cfg
	return nil
}

// SetConfigPath 设置配置文件路径
func SetConfigPath(path string) {
	configPath = path
}

// getDefaultConfigPath 获取默认配置文件路径
func getDefaultConfigPath() string {
	// 优先使用应用目录
	if execPath, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(execPath), "config", "browser_pool.json")
	}

	// 回退到当前工作目录
	if cwd, err := os.Getwd(); err == nil {
		return filepath.Join(cwd, "config", "browser_pool.json")
	}

	// 最后回退到临时目录
	return filepath.Join(os.TempDir(), "fuploader", "browser_pool.json")
}

// UpdateConfig 更新配置项
func UpdateConfig(updates map[string]interface{}) error {
	cfg := LoadPoolConfig()

	// 应用更新
	if v, ok := updates["max_browsers"].(float64); ok {
		cfg.MaxBrowsers = int(v)
	}
	if v, ok := updates["max_contexts_per_browser"].(float64); ok {
		cfg.MaxContextsPerBrowser = int(v)
	}
	if v, ok := updates["context_idle_timeout"].(float64); ok {
		cfg.ContextIdleTimeout = int(v)
	}
	if v, ok := updates["enable_health_check"].(bool); ok {
		cfg.EnableHealthCheck = v
	}
	if v, ok := updates["health_check_interval"].(float64); ok {
		cfg.HealthCheckInterval = int(v)
	}

	return SavePoolConfig(cfg)
}

// ResetToDefault 重置为默认配置
func ResetToDefault() error {
	cfg := DefaultPoolConfig
	return SavePoolConfig(&cfg)
}
