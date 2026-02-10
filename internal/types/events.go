package types

// Event 事件接口
// 所有事件类型都实现此接口，用于类型安全的EventBus
type Event interface {
	EventType() string
}

// UploadProgressEvent 上传进度事件
type UploadProgressEvent struct {
	TaskID   int    `json:"taskId"`
	Platform string `json:"platform"`
	Progress int    `json:"progress"`
	Message  string `json:"message"`
}

// EventType 返回事件类型
func (e UploadProgressEvent) EventType() string { return "upload_progress" }

// UploadCompleteEvent 上传完成事件
type UploadCompleteEvent struct {
	TaskID      int    `json:"taskId"`
	Platform    string `json:"platform"`
	PublishURL  string `json:"publishUrl"`
	CompletedAt string `json:"completedAt"`
}

// EventType 返回事件类型
func (e UploadCompleteEvent) EventType() string { return "upload_complete" }

// UploadErrorEvent 上传错误事件
type UploadErrorEvent struct {
	TaskID   int    `json:"taskId"`
	Platform string `json:"platform"`
	Error    string `json:"error"`
	CanRetry bool   `json:"canRetry"`
}

// EventType 返回事件类型
func (e UploadErrorEvent) EventType() string { return "upload_error" }

// LoginSuccessEvent 登录成功事件
type LoginSuccessEvent struct {
	AccountID int    `json:"accountId"`
	Platform  string `json:"platform"`
	Username  string `json:"username"`
}

// EventType 返回事件类型
func (e LoginSuccessEvent) EventType() string { return "login_success" }

// LoginErrorEvent 登录错误事件
type LoginErrorEvent struct {
	AccountID int    `json:"accountId"`
	Platform  string `json:"platform"`
	Error     string `json:"error"`
}

// EventType 返回事件类型
func (e LoginErrorEvent) EventType() string { return "login_error" }

// TaskStatusChangedEvent 任务状态变更事件
type TaskStatusChangedEvent struct {
	TaskID    int    `json:"taskId"`
	OldStatus string `json:"oldStatus"`
	NewStatus string `json:"newStatus"`
}

// EventType 返回事件类型
func (e TaskStatusChangedEvent) EventType() string { return "task_status_changed" }

// AccountStatusChangedEvent 账号状态变更事件
type AccountStatusChangedEvent struct {
	AccountID int `json:"accountId"`
	OldStatus int `json:"oldStatus"`
	NewStatus int `json:"newStatus"`
}

// EventType 返回事件类型
func (e AccountStatusChangedEvent) EventType() string { return "account_status_changed" }
