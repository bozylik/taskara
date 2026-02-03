package cluster

import "time"

// RetryBackoffStrategy defines the interface for calculating delays between retry attempts.
// Any custom backoff logic must implement this interface.
type RetryBackoffStrategy interface {
	// Next returns the duration to wait before the next attempt, given the current attempt number.
	Next(attempt int) time.Duration
}

// FixedBackoff provides a constant delay between all retry attempts.
type FixedBackoff struct {
	// Delay is the duration to wait between retries.
	Delay time.Duration
}

// Next returns the fixed delay regardless of the attempt number.
func (b FixedBackoff) Next(_ int) time.Duration { return b.Delay }

// LinearBackoff increases the delay by a constant step for each subsequent attempt.
type LinearBackoff struct {
	// Base is the initial delay duration.
	Base time.Duration
	
	// Step is the duration added to the delay after each attempt.
	Step time.Duration
}

// Next returns a delay that grows linearly: Base + (Step * attempt).
func (b LinearBackoff) Next(attempt int) time.Duration {
	return b.Base + (b.Step * time.Duration(attempt))
}

// ExponentialBackoff increases the delay exponentially, optionally capped by a maximum value.
type ExponentialBackoff struct {
	// Base is the starting delay duration.
	Base time.Duration

	// Multiplier is the factor by which the delay grows each attempt (e.g., 2.0).
	Multiplier float64

	// Max is the upper bound for the delay. If 0, no maximum is applied.
	Max time.Duration
}

// Next returns a delay that grows exponentially: Base * (Multiplier ^ attempt).
func (b ExponentialBackoff) Next(attempt int) time.Duration {
	if attempt <= 0 {
		return b.Base
	}

	delay := float64(b.Base)
	for i := 0; i < attempt; i++ {
		delay *= b.Multiplier
	}

	result := time.Duration(delay)

	if b.Max > 0 && result > b.Max {
		return b.Max
	}
	return result
}

// NewFixedBackoff creates a strategy that always returns the same delay.
func NewFixedBackoff(delay time.Duration) RetryBackoffStrategy {
	return FixedBackoff{delay}
}

// NewLinearBackoff creates a strategy that increases delay by a fixed step.
func NewLinearBackoff(base time.Duration, step time.Duration) RetryBackoffStrategy {
	return LinearBackoff{base, step}
}

// NewExponentialBackoff creates a strategy where delay grows by a multiplier.
// If multiplier is less than 1.0, it defaults to 2.0 to ensure growth.
func NewExponentialBackoff(base time.Duration, multiplier float64, max time.Duration) RetryBackoffStrategy {
	if multiplier < 1.0 {
		multiplier = 2.0
	}
	return ExponentialBackoff{base, multiplier, max}
}
