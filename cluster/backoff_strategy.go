package cluster

import "time"

type RetryBackoffStrategy interface {
	Next(attempt int) time.Duration
}

type FixedBackoff struct {
	Delay time.Duration
}

func (b FixedBackoff) Next(_ int) time.Duration { return b.Delay }

type LinearBackoff struct {
	Base time.Duration
	Step time.Duration
}

func (b LinearBackoff) Next(attempt int) time.Duration {
	return b.Base + (b.Step * time.Duration(attempt))
}

type ExponentialBackoff struct {
	Base       time.Duration
	Multiplier float64
	Max        time.Duration
}

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

func NewFixedBackoff(delay time.Duration) RetryBackoffStrategy {
	return FixedBackoff{delay}
}

func NewLinearBackoff(base time.Duration, step time.Duration) RetryBackoffStrategy {
	return LinearBackoff{base, step}
}

func NewExponentialBackoff(base time.Duration, multiplier float64, max time.Duration) RetryBackoffStrategy {
	if multiplier < 1.0 {
		multiplier = 2.0
	}
	return ExponentialBackoff{base, multiplier, max}
}
