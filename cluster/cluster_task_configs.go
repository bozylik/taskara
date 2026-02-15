package cluster

import "time"

type retryConfig struct {
	maxRetries     int
	retryBackoff   RetryBackoffStrategy
	jitter         bool
	retryIf        func(err error) bool
	retryMode      RetryMode
	currentAttempt int
}

type repeatConfig struct {
	repeatInterval time.Duration
	repeatUntil    time.Time
	repeatCount    int
}
