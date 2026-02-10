package types

// LogLevel 日志级别
type LogLevel string

const (
	LogLevelInfo    LogLevel = "info"
	LogLevelWarn    LogLevel = "warn"
	LogLevelError   LogLevel = "error"
	LogLevelDebug   LogLevel = "debug"
	LogLevelSuccess LogLevel = "success"
)

// SimpleLog 简洁日志条目
type SimpleLog struct {
	Date     string   `json:"date"`     // 日期，格式：2006/1/2
	Time     string   `json:"time"`     // 时间，格式：15:04:05
	Message  string   `json:"message"`  // 日志内容
	Platform string   `json:"platform"` // 平台标识 (bilibili/douyin/xiaohongshu等)
	Level    LogLevel `json:"level"`    // 日志级别
}

// LogQuery 日志查询参数
type LogQuery struct {
	Keyword  string   `json:"keyword"`  // 关键词搜索
	Limit    int      `json:"limit"`    // 返回条数，默认100
	Platform string   `json:"platform"` // 平台筛选，空字符串表示全部
	Level    LogLevel `json:"level"`    // 级别筛选，空字符串表示全部
}
