package types

// AppVersion 应用版本信息
type AppVersion struct {
	Version      string `json:"version"`
	BuildTime    string `json:"buildTime"`
	GoVersion    string `json:"goVersion"`
	WailsVersion string `json:"wailsVersion"`
}

// AppStatus 应用状态
type AppStatus struct {
	Initialized bool   `json:"initialized"`
	Error       string `json:"error"`
	Version     string `json:"version"`
}

// ProductLinkValidationResult 商品链接验证结果
type ProductLinkValidationResult struct {
	Valid bool   `json:"valid"`
	Title string `json:"title,omitempty"`
	Error string `json:"error,omitempty"`
}

// CacheStats 缓存统计
type CacheStats struct {
	Total   int `json:"total"`
	Valid   int `json:"valid"`
	Expired int `json:"expired"`
}

// Collection 合集信息
type Collection struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// CoverInfo 封面信息
type CoverInfo struct {
	ThumbnailPath string `json:"thumbnailPath"`
}

// StepResultData 步骤结果数据
type StepResultData struct {
	Type string `json:"type"`
}
