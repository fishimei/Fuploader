package kuaishou

import "time"

type Config struct {
	UploadTimeout        time.Duration
	PageLoadTimeout      time.Duration
	ElementWaitTimeout   time.Duration
	SubmitCheckTimeout   time.Duration
	MaxClickAttempts     int
	MaxUploadRetries     int
	MaxLoginWaitAttempts int
}

var defaultConfig = Config{
	UploadTimeout:        10 * time.Minute,
	PageLoadTimeout:      30 * time.Second,
	ElementWaitTimeout:   5 * time.Second,
	SubmitCheckTimeout:   60 * time.Second,
	MaxClickAttempts:     3,
	MaxUploadRetries:     60,
	MaxLoginWaitAttempts: 30,
}

func DefaultConfig() Config {
	return defaultConfig
}
