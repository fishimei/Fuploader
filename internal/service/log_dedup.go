package service

import (
	"Fuploader/internal/types"
	"regexp"
	"strings"
	"sync"
	"time"
)

// MergeRule 日志归并规则
type MergeRule struct {
	Pattern    *regexp.Regexp // 匹配模式
	TimeWindow time.Duration  // 时间窗口
	MaxCount   int            // 最大归并数
	ShowFirst  bool           // 是否显示第一条
	ShowLast   bool           // 是否显示最后一条
}

// MergedLog 归并后的日志
type MergedLog struct {
	types.SimpleLog
	IsMerged    bool   `json:"isMerged"`    // 是否是归并日志
	RepeatCount int    `json:"repeatCount"` // 重复次数
	StartTime   string `json:"startTime"`   // 开始时间
	EndTime     string `json:"endTime"`     // 结束时间
}

// logGroup 归并组（内部使用）
type logGroup struct {
	key      string
	level    string
	message  string
	firstLog types.SimpleLog
	lastLog  types.SimpleLog
	count    int
	lastTime time.Time
}

// LogDeduplicator 日志去重归并器
type LogDeduplicator struct {
	rules       []MergeRule
	groups      map[string]*logGroup
	mutex       sync.RWMutex
	maxWaitTime time.Duration // 最大等待时间，超时强制输出
}

// NewLogDeduplicator 创建日志归并器
func NewLogDeduplicator() *LogDeduplicator {
	return &LogDeduplicator{
		rules:       defaultRules(),
		groups:      make(map[string]*logGroup),
		maxWaitTime: 500 * time.Millisecond,
	}
}

// defaultRules 默认归并规则
func defaultRules() []MergeRule {
	return []MergeRule{
		// Cookie验证失败类日志
		{
			Pattern:    regexp.MustCompile(`(?i)cookie.*验证失败|playwright.*target closed|验证请求执行失败`),
			TimeWindow: 30 * time.Second,
			MaxCount:   100,
			ShowFirst:  true,
			ShowLast:   false,
		},
		// Cookie检测类日志
		{
			Pattern:    regexp.MustCompile(`(?i)检测到所有必需cookie|当前所有cookie`),
			TimeWindow: 10 * time.Second,
			MaxCount:   50,
			ShowFirst:  false,
			ShowLast:   false,
		},
		// 限流检查类日志
		{
			Pattern:    regexp.MustCompile(`(?i)限流检查|rate.?limit`),
			TimeWindow: 20 * time.Second,
			MaxCount:   50,
			ShowFirst:  true,
			ShowLast:   false,
		},
		// 重试类日志
		{
			Pattern:    regexp.MustCompile(`(?i)重试|retry|继续检测`),
			TimeWindow: 15 * time.Second,
			MaxCount:   30,
			ShowFirst:  false,
			ShowLast:   false,
		},
	}
}

// extractLevel 从日志消息中提取级别
func extractLevel(message string) string {
	message = strings.ToLower(message)
	switch {
	case strings.Contains(message, "[error]"):
		return "error"
	case strings.Contains(message, "[warn]"):
		return "warn"
	case strings.Contains(message, "[info]"):
		return "info"
	case strings.Contains(message, "[debug]"):
		return "debug"
	case strings.Contains(message, "[success]"):
		return "success"
	default:
		return "info"
	}
}

// normalizeMessage 归一化消息用于比较
func normalizeMessage(message string) string {
	// 移除时间戳、数字等变化部分
	re := regexp.MustCompile(`\d{2}:\d{2}:\d{2}|\d+次|第\d+次`)
	return re.ReplaceAllString(message, "")
}

// matchRule 匹配归并规则
func (d *LogDeduplicator) matchRule(message string) *MergeRule {
	for i := range d.rules {
		if d.rules[i].Pattern.MatchString(message) {
			return &d.rules[i]
		}
	}
	return nil
}

// generateKey 生成归并键（包含平台信息）
func generateKey(level, platform, normalizedMsg string) string {
	if platform != "" {
		return level + "|" + platform + "|" + normalizedMsg
	}
	return level + "|" + normalizedMsg
}

// Process 处理单条日志，返回需要立即输出的日志（可能是归并后的）
func (d *LogDeduplicator) Process(log types.SimpleLog) []MergedLog {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	level := extractLevel(log.Message)
	normalized := normalizeMessage(log.Message)
	key := generateKey(level, log.Platform, normalized)

	// 解析日志时间
	logTime, _ := time.Parse("15:04:05", log.Time)
	if logTime.IsZero() {
		logTime = time.Now()
	}

	// 检查是否匹配归并规则
	rule := d.matchRule(log.Message)
	if rule == nil {
		// 不匹配任何规则，直接输出
		return []MergedLog{{SimpleLog: log, IsMerged: false}}
	}

	// 检查是否已存在归并组
	group, exists := d.groups[key]
	if exists {
		// 检查时间窗口
		if logTime.Sub(group.lastTime) <= rule.TimeWindow && group.count < rule.MaxCount {
			// 在窗口内，累加
			group.count++
			group.lastTime = logTime
			group.lastLog = log
			return nil // 暂不输出
		}
		// 超出窗口或达到上限，先刷新旧组
		result := d.flushGroup(key)
		// 创建新组
		d.createGroup(key, log, logTime, rule)
		return result
	}

	// 创建新组
	d.createGroup(key, log, logTime, rule)
	return nil
}

// createGroup 创建新的归并组
func (d *LogDeduplicator) createGroup(key string, log types.SimpleLog, logTime time.Time, rule *MergeRule) {
	d.groups[key] = &logGroup{
		key:      key,
		level:    extractLevel(log.Message),
		message:  normalizeMessage(log.Message),
		firstLog: log,
		lastLog:  log,
		count:    1,
		lastTime: logTime,
	}
}

// flushGroup 刷新指定归并组
func (d *LogDeduplicator) flushGroup(key string) []MergedLog {
	group, exists := d.groups[key]
	if !exists || group.count == 0 {
		return nil
	}

	delete(d.groups, key)

	// 查找规则
	rule := d.matchRule(group.firstLog.Message)
	if rule == nil {
		return []MergedLog{{SimpleLog: group.firstLog, IsMerged: false}}
	}

	var results []MergedLog

	// 根据配置决定是否显示第一条
	if rule.ShowFirst && group.count > 0 {
		results = append(results, MergedLog{
			SimpleLog:   group.firstLog,
			IsMerged:    false,
			RepeatCount: 1,
		})
	}

	// 如果有多条，添加归并记录
	if group.count > 1 {
		mergedMsg := group.firstLog.Message
		if !rule.ShowFirst {
			// 如果不显示第一条，用第一条作为归并消息
			results = append(results, MergedLog{
				SimpleLog: types.SimpleLog{
					Date:    group.firstLog.Date,
					Time:    group.firstLog.Time,
					Message: mergedMsg,
				},
				IsMerged:    true,
				RepeatCount: group.count,
				StartTime:   group.firstLog.Time,
				EndTime:     group.lastLog.Time,
			})
		} else {
			// 显示第一条后，添加归并提示
			results = append(results, MergedLog{
				SimpleLog: types.SimpleLog{
					Date:    group.firstLog.Date,
					Time:    group.firstLog.Time,
					Message: "  ↳ 该消息在后续 " + rule.TimeWindow.String() + " 内重复出现 " + string(rune('0'+group.count)) + " 次 (" + group.firstLog.Time + " ~ " + group.lastLog.Time + ")",
				},
				IsMerged:    true,
				RepeatCount: group.count,
				StartTime:   group.firstLog.Time,
				EndTime:     group.lastLog.Time,
			})
		}
	}

	// 根据配置决定是否显示最后一条
	if rule.ShowLast && group.count > 1 {
		results = append(results, MergedLog{
			SimpleLog:   group.lastLog,
			IsMerged:    false,
			RepeatCount: 1,
		})
	}

	return results
}

// FlushAll 刷新所有待归并的日志
func (d *LogDeduplicator) FlushAll() []MergedLog {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	var results []MergedLog
	for key := range d.groups {
		if groupResults := d.flushGroup(key); len(groupResults) > 0 {
			results = append(results, groupResults...)
		}
	}
	return results
}

// GetPendingCount 获取待归并的日志组数量
func (d *LogDeduplicator) GetPendingCount() int {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return len(d.groups)
}
