package baijiahao

import "time"

type Config struct {
	UploadTimeout        time.Duration
	PageLoadTimeout      time.Duration
	ElementWaitTimeout   time.Duration
	SubmitCheckTimeout   time.Duration
	MaxPublishRetries    int
	TitleMinLength       int
	TitleMaxLength       int
	UploadCheckInterval  time.Duration
}

var defaultConfig = Config{
	UploadTimeout:        10 * time.Minute,
	PageLoadTimeout:      30 * time.Second,
	ElementWaitTimeout:   5 * time.Second,
	SubmitCheckTimeout:   60 * time.Second,
	MaxPublishRetries:    3,
	TitleMinLength:       8,
	TitleMaxLength:       30,
	UploadCheckInterval:  2 * time.Second,
}

func DefaultConfig() Config {
	return defaultConfig
}
