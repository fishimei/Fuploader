package tiktok

import "time"

type Config struct {
	UploadTimeout       time.Duration
	PageLoadTimeout     time.Duration
	ElementWaitTimeout  time.Duration
	SubmitCheckTimeout  time.Duration
	MaxPublishRetries   int
	MaxUploadRetries    int
	TitleMaxLength      int
	UploadCheckInterval time.Duration
}

var defaultConfig = Config{
	UploadTimeout:       10 * time.Minute,
	PageLoadTimeout:     30 * time.Second,
	ElementWaitTimeout:  10 * time.Second,
	SubmitCheckTimeout:  120 * time.Second,
	MaxPublishRetries:   60,
	MaxUploadRetries:    60,
	TitleMaxLength:      150,
	UploadCheckInterval: 2 * time.Second,
}

func DefaultConfig() Config {
	return defaultConfig
}
