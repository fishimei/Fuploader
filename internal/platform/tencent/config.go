package tencent

import "time"

type Config struct {
	UploadTimeout        time.Duration
	PageLoadTimeout      time.Duration
	ElementWaitTimeout   time.Duration
	SubmitCheckTimeout   time.Duration
	MaxPublishRetries    int
	MaxUploadRetries     int
	ShortTitleMinLength  int
	ShortTitleMaxLength  int
	UploadCheckInterval  time.Duration
}

var defaultConfig = Config{
	UploadTimeout:        10 * time.Minute,
	PageLoadTimeout:      30 * time.Second,
	ElementWaitTimeout:   5 * time.Second,
	SubmitCheckTimeout:   30 * time.Second,
	MaxPublishRetries:    3,
	MaxUploadRetries:     3,
	ShortTitleMinLength:  6,
	ShortTitleMaxLength:  16,
	UploadCheckInterval:  2 * time.Second,
}

func DefaultConfig() Config {
	return defaultConfig
}
