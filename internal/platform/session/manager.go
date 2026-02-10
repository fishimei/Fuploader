package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"Fuploader/internal/types"
	"Fuploader/internal/utils"
)

// Session 会话信息
type Session struct {
	ID           string                 `json:"id"`
	Platform     string                 `json:"platform"`
	AccountID    uint                   `json:"account_id"`
	Cookies      []Cookie               `json:"cookies"`
	StorageState map[string]interface{} `json:"storage_state"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	ExpiresAt    time.Time              `json:"expires_at"`
	LastVerified time.Time              `json:"last_verified"`
	IsValid      bool                   `json:"is_valid"`
	Metadata     map[string]string      `json:"metadata"`
}

// Cookie Cookie 信息
type Cookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain"`
	Path     string `json:"path"`
	Expires  int64  `json:"expires"`
	HttpOnly bool   `json:"httpOnly"`
	Secure   bool   `json:"secure"`
	SameSite string `json:"sameSite"`
}

// Validator 会话验证器接口
type Validator interface {
	Validate(ctx context.Context, session *Session) (bool, error)
	Platform() string
}

// Manager 会话管理器
type Manager struct {
	cache      map[string]*Session
	mutex      sync.RWMutex
	validators map[string]Validator
	cookieDir  string
}

// NewManager 创建会话管理器
func NewManager(cookieDir string) *Manager {
	m := &Manager{
		cache:      make(map[string]*Session),
		validators: make(map[string]Validator),
		cookieDir:  cookieDir,
	}

	// 启动定期清理任务
	go m.cleanupLoop()

	return m
}

// RegisterValidator 注册平台验证器
func (m *Manager) RegisterValidator(validator Validator) {
	m.validators[validator.Platform()] = validator
}

// LoadSession 加载会话
func (m *Manager) LoadSession(ctx context.Context, accountID uint, platform string) (*Session, error) {
	sessionID := fmt.Sprintf("%s_%d", platform, accountID)

	// 1. 检查缓存
	m.mutex.RLock()
	if session, ok := m.cache[sessionID]; ok && session.IsValid {
		// 检查是否过期
		if time.Now().Before(session.ExpiresAt) {
			m.mutex.RUnlock()
			return session, nil
		}
	}
	m.mutex.RUnlock()

	// 2. 从文件加载
	session, err := m.loadFromFile(accountID, platform)
	if err != nil {
		return nil, err
	}

	// 3. 验证会话
	if validator, ok := m.validators[platform]; ok {
		isValid, err := validator.Validate(ctx, session)
		if err != nil {
			return nil, fmt.Errorf("validate session failed: %w", err)
		}
		session.IsValid = isValid
		session.LastVerified = time.Now()
	}

	// 4. 更新缓存
	m.mutex.Lock()
	m.cache[sessionID] = session
	m.mutex.Unlock()

	return session, nil
}

// SaveSession 保存会话
func (m *Manager) SaveSession(session *Session) error {
	session.UpdatedAt = time.Now()

	// 设置过期时间（默认7天）
	if session.ExpiresAt.IsZero() {
		session.ExpiresAt = time.Now().Add(7 * 24 * time.Hour)
	}

	// 保存到文件
	if err := m.saveToFile(session); err != nil {
		return fmt.Errorf("save to file failed: %w", err)
	}

	// 更新缓存
	sessionID := fmt.Sprintf("%s_%d", session.Platform, session.AccountID)
	m.mutex.Lock()
	m.cache[sessionID] = session
	m.mutex.Unlock()

	return nil
}

// InvalidateSession 使会话失效
func (m *Manager) InvalidateSession(accountID uint, platform string) error {
	sessionID := fmt.Sprintf("%s_%d", platform, accountID)

	m.mutex.Lock()
	delete(m.cache, sessionID)
	m.mutex.Unlock()

	// 删除文件
	cookiePath := m.getCookiePath(accountID, platform)
	if err := os.Remove(cookiePath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// RefreshSession 刷新会话
func (m *Manager) RefreshSession(accountID uint, platform string, newCookies []Cookie) error {
	sessionID := fmt.Sprintf("%s_%d", platform, accountID)

	m.mutex.RLock()
	session, ok := m.cache[sessionID]
	m.mutex.RUnlock()

	if !ok {
		// 从文件加载
		var err error
		session, err = m.loadFromFile(accountID, platform)
		if err != nil {
			return err
		}
	}

	session.Cookies = newCookies
	session.UpdatedAt = time.Now()
	session.ExpiresAt = time.Now().Add(7 * 24 * time.Hour) // 延长7天
	session.IsValid = true

	return m.SaveSession(session)
}

// GetSessionStatus 获取会话状态
func (m *Manager) GetSessionStatus(accountID uint, platform string) (*Session, error) {
	sessionID := fmt.Sprintf("%s_%d", platform, accountID)

	m.mutex.RLock()
	if session, ok := m.cache[sessionID]; ok {
		m.mutex.RUnlock()
		return session, nil
	}
	m.mutex.RUnlock()

	return m.loadFromFile(accountID, platform)
}

// loadFromFile 从文件加载会话
func (m *Manager) loadFromFile(accountID uint, platform string) (*Session, error) {
	cookiePath := m.getCookiePath(accountID, platform)

	data, err := os.ReadFile(cookiePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session not found")
		}
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// saveToFile 保存到文件
func (m *Manager) saveToFile(session *Session) error {
	cookiePath := m.getCookiePath(session.AccountID, session.Platform)

	// 确保目录存在
	dir := filepath.Dir(cookiePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cookiePath, data, 0644)
}

// getCookiePath 获取 Cookie 文件路径
func (m *Manager) getCookiePath(accountID uint, platform string) string {
	return filepath.Join(m.cookieDir, fmt.Sprintf("%s_%d.json", platform, accountID))
}

// cleanupLoop 定期清理过期会话
func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		m.cleanup()
	}
}

// cleanup 清理过期会话
func (m *Manager) cleanup() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	for id, session := range m.cache {
		if now.After(session.ExpiresAt) || !session.IsValid {
			delete(m.cache, id)
			utils.Info(fmt.Sprintf("[-] 清理过期会话 - ID: %s", id))
		}
	}
}

// ClearCache 清空缓存
func (m *Manager) ClearCache() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.cache = make(map[string]*Session)
	utils.Info("[-] 会话缓存已清空")
}

// GetCacheStats 获取缓存统计
func (m *Manager) GetCacheStats() *types.CacheStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	validCount := 0
	expiredCount := 0

	now := time.Now()
	for _, session := range m.cache {
		if session.IsValid && now.Before(session.ExpiresAt) {
			validCount++
		} else {
			expiredCount++
		}
	}

	return &types.CacheStats{
		Total:   len(m.cache),
		Valid:   validCount,
		Expired: expiredCount,
	}
}
