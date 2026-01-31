package cluster

type retryConfig struct {
	maxRetries     int
	retryBackoff   RetryBackoffStrategy
	jitter         bool
	retryIf        func(err error) bool
	retryMode      RetryMode
	currentAttempt int
}
