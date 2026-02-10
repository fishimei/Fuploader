package types

import "context"

// VideoTask 视频任务
type VideoTask struct {
	Platform     string // 平台名称
	VideoPath    string
	Title        string
	Description  string
	Tags         []string
	Thumbnail    string // 封面路径
	ScheduleTime *string
	IsDraft      bool   // 是否保存为草稿
	Location     string // 地理位置
	SyncToutiao  bool   // 同步到今日头条
	SyncXigua    bool   // 同步到西瓜视频
	ShortTitle   string // 视频号短标题
	IsOriginal   bool   // 是否声明原创
	OriginalType string // 原创类型
	Collection   string // 合集名称
	ProductLink  string // 商品链接（抖音）
	ProductTitle string // 商品短标题（抖音）
}

// Uploader 上传器接口
type Uploader interface {
	ValidateCookie(ctx context.Context) (bool, error)
	Upload(ctx context.Context, task *VideoTask) error
	Login() error
	Platform() string
}

// PlatformFields 平台特定字段
type PlatformFields struct {
	Title        string `json:"title"`
	Collection   string `json:"collection"`
	ShortTitle   string `json:"shortTitle"`
	IsOriginal   bool   `json:"isOriginal"`
	OriginalType string `json:"originalType"`
	Location     string `json:"location"`
	Thumbnail    string `json:"thumbnail"`
	SyncToutiao  bool   `json:"syncToutiao"`
	SyncXigua    bool   `json:"syncXigua"`
	IsDraft      bool   `json:"isDraft"`
}

// CommonMetadata 通用元数据
type CommonMetadata struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// UploadTaskMetadata 上传任务元数据
type UploadTaskMetadata struct {
	Common    CommonMetadata            `json:"common"`
	Platforms map[string]PlatformFields `json:"platforms"`
}
