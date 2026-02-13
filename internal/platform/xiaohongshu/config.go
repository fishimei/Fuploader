package xiaohongshu

import "time"

type Config struct {
	UploadTimeout        time.Duration
	PageLoadTimeout      time.Duration
	ElementWaitTimeout   time.Duration
	SubmitCheckTimeout   time.Duration
	MaxPublishRetries    int
	TitleMaxLength       int
	UploadCheckInterval  time.Duration
	MaxLoginWaitAttempts int
}

var defaultConfig = Config{
	UploadTimeout:        5 * time.Minute,
	PageLoadTimeout:      30 * time.Second,
	ElementWaitTimeout:   5 * time.Second,
	SubmitCheckTimeout:   30 * time.Second,
	MaxPublishRetries:    20,
	TitleMaxLength:       30,
	UploadCheckInterval:  500 * time.Millisecond,
	MaxLoginWaitAttempts: 30,
}

func DefaultConfig() Config {
	return defaultConfig
}
