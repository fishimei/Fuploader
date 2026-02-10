// Package retry 提供全面的重试机制实现
package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"Fuploader/internal/utils"
)

// RetryStrategy 重试策略类型
type RetryStrategy string

const (
	// ExponentialBackoff 指数退避策略
	ExponentialBackoff RetryStrategy = "exponential_backoff"
	// FixedInterval 固定间隔策略
	FixedInterval RetryStrategy = "fixed_interval"
	// RandomDelay 随机延迟策略
	RandomDelay RetryStrategy = "random_delay"
	// LinearBackoff 线性退避策略
	LinearBackoff RetryStrategy = "linear_backoff"
)

// RetryCondition 重试条件函数
type RetryCondition func(error) bool

// RetryCallback 重试回调函数
type RetryCallback func(attempt int, delay time.Duration, err error)

// Config 重试配置
type Config struct {
	// 基础配置
	MaxRetries   int           // 最大重试次数
	InitialDelay time.Duration // 初始延迟
	MaxDelay     time.Duration // 最大延迟
	TotalTimeout time.Duration // 总超时时间

	// 策略配置
	Strategy      RetryStrategy // 重试策略
	BackoffFactor float64       // 退避因子（用于指数退避）
	Jitter        bool          // 是否启用抖动
	JitterFactor  float64       // 抖动因子 (0.0 - 1.0)

	// 条件配置
	RetryableErrors []string       // 可重试的错误类型
	RetryCondition  RetryCondition // 自定义重试条件

	// 回调配置
	OnRetry   RetryCallback // 重试时回调
	OnSuccess func()        // 成功时回调
	OnFailure func(error)   // 最终失败时回调

	// 限流配置
	RateLimit *RateLimitConfig // 限流配置
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	RequestsPerSecond float64       // 每秒请求数
	BurstSize         int           // 突发请求数
	CooldownPeriod    time.Duration // 冷却期
}

// DefaultConfig 默认重试配置
func DefaultConfig() *Config {
	return &Config{
		MaxRetries:      3,
		InitialDelay:    2 * time.Second,
		MaxDelay:        30 * time.Second,
		TotalTimeout:    5 * time.Minute,
		Strategy:        ExponentialBackoff,
		BackoffFactor:   2.0,
		Jitter:          true,
		JitterFactor:    0.1,
		RetryableErrors: []string{},
	}
}

// Retry 重试器
type Retry struct {
	config    *Config
	attempts  int32
	successes int32
	failures  int32
}

// NewRetry 创建重试器
func NewRetry(config *Config) *Retry {
	if config == nil {
		config = DefaultConfig()
	}
	return &Retry{config: config}
}

// Do 执行带重试的操作
func (r *Retry) Do(ctx context.Context, operation func() error) error {
	if r.config.TotalTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.config.TotalTimeout)
		defer cancel()
	}

	var lastErr error
	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := r.calculateDelay(attempt)
			if r.config.OnRetry != nil {
				r.config.OnRetry(attempt, delay, lastErr)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := operation()
		if err == nil {
			atomic.AddInt32(&r.successes, 1)
			if r.config.OnSuccess != nil {
				r.config.OnSuccess()
			}
			return nil
		}

		lastErr = err
		atomic.AddInt32(&r.attempts, 1)

		if !r.shouldRetry(err) {
			break
		}
	}

	atomic.AddInt32(&r.failures, 1)
	if r.config.OnFailure != nil {
		r.config.OnFailure(lastErr)
	}
	return lastErr
}

// DoWithResult 执行带重试的操作并返回结果
func DoWithResult[T any](ctx context.Context, config *Config, operation func() (T, error)) (T, error) {
	var result T
	r := NewRetry(config)
	err := r.Do(ctx, func() error {
		var err error
		result, err = operation()
		return err
	})
	return result, err
}

// calculateDelay 计算重试延迟
func (r *Retry) calculateDelay(attempt int) time.Duration {
	var delay time.Duration

	switch r.config.Strategy {
	case ExponentialBackoff:
		delay = time.Duration(float64(r.config.InitialDelay) * math.Pow(r.config.BackoffFactor, float64(attempt-1)))
	case LinearBackoff:
		delay = time.Duration(int64(r.config.InitialDelay) * int64(attempt))
	case FixedInterval:
		delay = r.config.InitialDelay
	case RandomDelay:
		delay = time.Duration(rand.Int63n(int64(r.config.InitialDelay)) + int64(r.config.InitialDelay)/2)
	default:
		delay = r.config.InitialDelay
	}

	if delay > r.config.MaxDelay {
		delay = r.config.MaxDelay
	}

	if r.config.Jitter {
		jitter := time.Duration(float64(delay) * r.config.JitterFactor * (rand.Float64()*2 - 1))
		delay += jitter
	}

	return delay
}

// shouldRetry 判断是否应重试
func (r *Retry) shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	if r.config.RetryCondition != nil {
		return r.config.RetryCondition(err)
	}

	return true
}

// GetStats 获取统计信息
func (r *Retry) GetStats() map[string]int32 {
	return map[string]int32{
		"attempts":  atomic.LoadInt32(&r.attempts),
		"successes": atomic.LoadInt32(&r.successes),
		"failures":  atomic.LoadInt32(&r.failures),
	}
}

// Reset 重置统计
func (r *Retry) Reset() {
	atomic.StoreInt32(&r.attempts, 0)
	atomic.StoreInt32(&r.successes, 0)
	atomic.StoreInt32(&r.failures, 0)
}

// RetryWithContext 使用上下文进行重试的便捷函数
func RetryWithContext(ctx context.Context, maxRetries int, operation func() error) error {
	config := DefaultConfig()
	config.MaxRetries = maxRetries
	r := NewRetry(config)
	return r.Do(ctx, operation)
}

// RetryWithBackoff 使用指数退避进行重试的便捷函数
func RetryWithBackoff(maxRetries int, initialDelay time.Duration, operation func() error) error {
	config := DefaultConfig()
	config.MaxRetries = maxRetries
	config.InitialDelay = initialDelay
	config.Strategy = ExponentialBackoff
	r := NewRetry(config)
	return r.Do(context.Background(), operation)
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	maxFailures  int32
	resetTimeout time.Duration
	failureCount int32
	lastFailure  time.Time
	state        int32 // 0: closed, 1: open, 2: half-open
	mutex        sync.RWMutex
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(maxFailures int32, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
	}
}

// Execute 执行操作，带熔断保护
func (cb *CircuitBreaker) Execute(operation func() error) error {
	if !cb.canExecute() {
		return fmt.Errorf("circuit breaker is open")
	}

	err := operation()
	if err != nil {
		cb.recordFailure()
		return err
	}

	cb.recordSuccess()
	return nil
}

// canExecute 检查是否可以执行
func (cb *CircuitBreaker) canExecute() bool {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	state := atomic.LoadInt32(&cb.state)
	if state == 0 { // closed
		return true
	}

	if state == 1 { // open
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			atomic.StoreInt32(&cb.state, 2) // half-open
			return true
		}
		return false
	}

	return true // half-open
}

// recordFailure 记录失败
func (cb *CircuitBreaker) recordFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.lastFailure = time.Now()
	count := atomic.AddInt32(&cb.failureCount, 1)

	if count >= cb.maxFailures {
		atomic.StoreInt32(&cb.state, 1) // open
		utils.Warn(fmt.Sprintf("[-] 熔断器开启，失败次数: %d", count))
	}
}

// recordSuccess 记录成功
func (cb *CircuitBreaker) recordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	atomic.StoreInt32(&cb.failureCount, 0)
	atomic.StoreInt32(&cb.state, 0) // closed
}

// GetState 获取熔断器状态
func (cb *CircuitBreaker) GetState() string {
	state := atomic.LoadInt32(&cb.state)
	switch state {
	case 0:
		return "closed"
	case 1:
		return "open"
	case 2:
		return "half-open"
	default:
		return "unknown"
	}
}
