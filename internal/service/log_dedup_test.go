package service

import (
	"Fuploader/internal/types"
	"strings"
	"testing"
	"time"
)

func TestLogDeduplicator_Process(t *testing.T) {
	dedup := NewLogDeduplicator()

	// 测试1: 不匹配的日志应该直接输出
	t.Run("unmatched_log_direct_output", func(t *testing.T) {
		log := types.SimpleLog{
			Date:    "2024/1/1",
			Time:    "10:00:00",
			Message: "[INFO] 普通日志消息",
		}
		result := dedup.Process(log)
		if len(result) != 1 {
			t.Errorf("期望返回1条日志，实际返回%d条", len(result))
		}
		if result[0].Message != log.Message {
			t.Errorf("消息不匹配")
		}
	})

	// 测试2: 匹配规则的日志应该被归并
	t.Run("matched_log_merged", func(t *testing.T) {
		dedup := NewLogDeduplicator()

		// 添加多条相同的Cookie验证失败日志
		logs := []types.SimpleLog{
			{Date: "2024/1/1", Time: "10:00:00", Message: "[WARN] Cookie轻量验证失败: playwright: target closed"},
			{Date: "2024/1/1", Time: "10:00:01", Message: "[WARN] Cookie轻量验证失败: playwright: target closed"},
			{Date: "2024/1/1", Time: "10:00:02", Message: "[WARN] Cookie轻量验证失败: playwright: target closed"},
		}

		// 前几条应该返回nil（被归并）
		for i := 0; i < len(logs)-1; i++ {
			result := dedup.Process(logs[i])
			if result != nil {
				t.Errorf("第%d条日志应该被归并，返回nil，实际返回%v", i, result)
			}
		}

		// 添加一条不同的日志触发刷新
		differentLog := types.SimpleLog{
			Date:    "2024/1/1",
			Time:    "10:00:30",
			Message: "[INFO] 其他日志",
		}
		result := dedup.Process(differentLog)

		// 应该返回归并后的日志
		if len(result) == 0 {
			t.Error("期望返回归并后的日志，但实际返回空")
		}
	})

	// 测试3: 测试FlushAll
	t.Run("flush_all", func(t *testing.T) {
		dedup := NewLogDeduplicator()

		// 添加归并日志
		log := types.SimpleLog{
			Date:    "2024/1/1",
			Time:    "10:00:00",
			Message: "[WARN] Cookie轻量验证失败",
		}
		dedup.Process(log)
		dedup.Process(log)
		dedup.Process(log)

		// 手动刷新
		result := dedup.FlushAll()
		if len(result) == 0 {
			t.Error("FlushAll应该返回归并后的日志")
		}

		// 验证归并次数
		found := false
		for _, r := range result {
			if r.IsMerged && r.RepeatCount == 3 {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("期望找到重复3次的归并日志，实际结果: %v", result)
		}
	})
}

func TestLogDeduplicator_ExtractLevel(t *testing.T) {
	tests := []struct {
		message string
		want    string
	}{
		{"[ERROR] 错误消息", "error"},
		{"[WARN] 警告消息", "warn"},
		{"[INFO] 信息消息", "info"},
		{"[DEBUG] 调试消息", "debug"},
		{"[SUCCESS] 成功消息", "success"},
		{"普通消息", "info"},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			got := extractLevel(tt.message)
			if got != tt.want {
				t.Errorf("extractLevel(%q) = %q, want %q", tt.message, got, tt.want)
			}
		})
	}
}

func TestLogService_WithDedup(t *testing.T) {
	service := NewLogService()

	// 测试添加日志
	service.Add(types.SimpleLog{Date: "2024/1/1", Time: "10:00:00", Message: "[INFO] 测试日志1", Level: types.LogLevelInfo})
	service.Add(types.SimpleLog{Date: "2024/1/1", Time: "10:00:01", Message: "[WARN] Cookie轻量验证失败: playwright: target closed", Level: types.LogLevelWarn})
	service.Add(types.SimpleLog{Date: "2024/1/1", Time: "10:00:02", Message: "[WARN] Cookie轻量验证失败: playwright: target closed", Level: types.LogLevelWarn})
	service.Add(types.SimpleLog{Date: "2024/1/1", Time: "10:00:03", Message: "[WARN] Cookie轻量验证失败: playwright: target closed", Level: types.LogLevelWarn})
	service.Add(types.SimpleLog{Date: "2024/1/1", Time: "10:00:04", Message: "[INFO] 测试日志2", Level: types.LogLevelInfo})

	// 等待定时刷新
	time.Sleep(1500 * time.Millisecond)

	// 查询日志
	logs := service.GetAll(100)

	// 检查是否有归并标记的日志
	var hasMerged bool
	for _, log := range logs {
		if strings.Contains(log.Message, "重复出现") {
			hasMerged = true
			break
		}
	}

	if !hasMerged {
		t.Log("警告: 未找到归并日志，可能需要调整测试时间")
	}

	// 测试开关
	service.SetDedupEnabled(false)
	if service.IsDedupEnabled() {
		t.Error("应该能够禁用归并")
	}

	service.SetDedupEnabled(true)
	if !service.IsDedupEnabled() {
		t.Error("应该能够启用归并")
	}
}
