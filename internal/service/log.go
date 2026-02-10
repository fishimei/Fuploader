package service

import (
	"Fuploader/internal/types"
	"strings"
	"sync"
	"time"
)

// LogService 日志服务
type LogService struct {
	logs         []types.SimpleLog
	mutex        sync.RWMutex
	limit        int                // 最大保留日志条数
	deduplicator *LogDeduplicator   // 日志归并器
	enableDedup  bool               // 是否启用归并
}

// NewLogService 创建日志服务
func NewLogService() *LogService {
	s := &LogService{
		logs:         make([]types.SimpleLog, 0, 500),
		limit:        500,
		deduplicator: NewLogDeduplicator(),
		enableDedup:  true, // 默认启用归并
	}
	// 启动定时刷新协程
	go s.startFlushLoop()
	return s
}

// startFlushLoop 启动定时刷新循环
func (s *LogService) startFlushLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.flushPending()
	}
}

// flushPending 刷新待归并的日志
func (s *LogService) flushPending() {
	if !s.enableDedup {
		return
	}

	mergedLogs := s.deduplicator.FlushAll()
	if len(mergedLogs) == 0 {
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, merged := range mergedLogs {
		s.logs = append(s.logs, merged.SimpleLog)
	}

	// 超过限制时，移除最旧的日志
	if len(s.logs) > s.limit {
		s.logs = s.logs[len(s.logs)-s.limit:]
	}
}

// Add 添加日志（实现 LogServiceInterface 接口）
func (s *LogService) Add(log types.SimpleLog) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 如果启用了归并，使用归并器处理
	if s.enableDedup {
		mergedLogs := s.deduplicator.Process(log)
		for _, merged := range mergedLogs {
			s.logs = append(s.logs, merged.SimpleLog)
		}
	} else {
		s.logs = append(s.logs, log)
	}

	// 超过限制时，移除最旧的日志
	if len(s.logs) > s.limit {
		s.logs = s.logs[len(s.logs)-s.limit:]
	}
}

// Query 查询日志
func (s *LogService) Query(query types.LogQuery) []types.SimpleLog {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	limit := query.Limit
	if limit <= 0 {
		limit = 100
	}

	result := make([]types.SimpleLog, 0, limit)

	// 倒序遍历，最新的在前面
	for i := len(s.logs) - 1; i >= 0 && len(result) < limit; i-- {
		log := s.logs[i]

		// 关键词筛选
		if query.Keyword != "" && !strings.Contains(log.Message, query.Keyword) {
			continue
		}

		// 平台筛选
		if query.Platform != "" && log.Platform != query.Platform {
			continue
		}

		// 级别筛选
		if query.Level != "" && log.Level != query.Level {
			continue
		}

		result = append(result, log)
	}

	return result
}

// GetAll 获取所有日志
func (s *LogService) GetAll(limit int) []types.SimpleLog {
	if limit <= 0 {
		limit = 100
	}

	return s.Query(types.LogQuery{Limit: limit})
}

// Clear 清空日志
func (s *LogService) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.logs = make([]types.SimpleLog, 0, s.limit)
	// 同时清空归并器
	if s.deduplicator != nil {
		s.deduplicator.FlushAll()
	}
}

// Count 获取日志数量
func (s *LogService) Count() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return len(s.logs)
}

// SetDedupEnabled 设置是否启用日志归并
func (s *LogService) SetDedupEnabled(enabled bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 如果关闭归并，先刷新所有待归并的日志
	if !enabled && s.enableDedup {
		s.flushPending()
	}

	s.enableDedup = enabled
}

// IsDedupEnabled 获取归并是否启用
func (s *LogService) IsDedupEnabled() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.enableDedup
}

// GetPendingDedupCount 获取待归并的日志组数量
func (s *LogService) GetPendingDedupCount() int {
	if s.deduplicator == nil {
		return 0
	}
	return s.deduplicator.GetPendingCount()
}

// GetPlatforms 获取所有有日志的平台列表
func (s *LogService) GetPlatforms() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	platformMap := make(map[string]bool)
	for _, log := range s.logs {
		if log.Platform != "" {
			platformMap[log.Platform] = true
		}
	}

	platforms := make([]string, 0, len(platformMap))
	for platform := range platformMap {
		platforms = append(platforms, platform)
	}
	return platforms
}
