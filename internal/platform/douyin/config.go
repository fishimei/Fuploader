package douyin

import "time"

type Config struct {
	UploadTimeout        time.Duration
	PageLoadTimeout      time.Duration
	ElementWaitTimeout   time.Duration
	SubmitCheckTimeout   time.Duration
	MaxPublishRetries    int
	TitleMaxLength       int
	ShortTitleMaxLength  int
	UploadCheckInterval  time.Duration
}

var defaultConfig = Config{
	UploadTimeout:        10 * time.Minute,
	PageLoadTimeout:      30 * time.Second,
	ElementWaitTimeout:   5 * time.Second,
	SubmitCheckTimeout:   100 * time.Second,
	MaxPublishRetries:    20,
	TitleMaxLength:       30,
	ShortTitleMaxLength:  10,
	UploadCheckInterval:  2 * time.Second,
}

func DefaultConfig() Config {
	return defaultConfig
}
