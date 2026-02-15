package cluster

import (
	"fmt"
	"github.com/bozylik/taskara/task"
	"time"
)

// ClusterTaskBuilderInterface provides a fluent API for configuring and submitting a task to the cluster.
// It allows setting execution parameters such as start time, timeout, priority, and lifecycle callbacks.
// The configuration must be finalized by calling the Submit method.
type ClusterTaskBuilderInterface interface {
	// WithStartTime schedules the task to run at a specific time.
	// If the time is in the past or time.Now(), the task will be executed as soon as a worker is available.
	WithStartTime(st time.Time) ClusterTaskBuilderInterface

	// WithTimeout sets a maximum execution time for the task.
	// If the task exceeds this duration, its ctx will be cancelled, and the task will be marked as timed out.
	WithTimeout(tm time.Duration) ClusterTaskBuilderInterface

	// WithPriority sets the task's priority.
	// Higher values (or lower, depending on your heap logic—usually higher) will move the task to the front of the queue.
	WithPriority(p int) ClusterTaskBuilderInterface

	// WithRetry sets the maximum number of times a task will be retried upon failure.
	// Default is 0 (no retries).
	WithRetry(r int) ClusterTaskBuilderInterface

	// WithBackoffStrategy defines the delay logic between retry attempts.
	// Default is FixedBackoff with 0s delay if not specified.
	WithBackoffStrategy(strategy RetryBackoffStrategy) ClusterTaskBuilderInterface

	// WithJitter adds a small random variation to the retry delay to prevent thundering herd problems.
	WithJitter() ClusterTaskBuilderInterface

	// RetryIf registers a predicate function to decide whether a retry should be attempted based on the error.
	// If the function returns true, the task is retried; otherwise, it fails immediately.
	RetryIf(func(err error) bool) ClusterTaskBuilderInterface

	// WithRetryMode defines where the retry delay happens.
	// Default is Requeue (returns the task to the queue).
	WithRetryMode(mode RetryMode) ClusterTaskBuilderInterface

	WithRepeatInterval(interval time.Duration) ClusterTaskBuilderInterface
	RepeatUntil(until time.Time) ClusterTaskBuilderInterface
	WithRepeatCount(n int) ClusterTaskBuilderInterface

	WithCache(ttl time.Duration) ClusterTaskBuilderInterface

	// OnComplete registers a callback that will be invoked once the task finishes its execution,
	// regardless of whether it succeeded, failed, or was cancelled.
	// The callback receives the task's unique ID, the resulting value, and any error encountered.
	OnComplete(fn func(id string, val any, err error)) ClusterTaskBuilderInterface

	// OnFailure registers a callback that will be invoked only if the task ends with an error,
	// a panic, or is cancelled.
	// The callback receives the task's unique ID and the error that caused the failure.
	OnFailure(fn func(id string, err error)) ClusterTaskBuilderInterface

	// Submit - the final method in the chain.
	// It validates the task, generates an ID (if empty), and pushes the task into the scheduler.
	// Returns an error if a task with the same ID is already running or managed by the cluster.
	Submit() (string, error)
}

type clusterTaskBuilder struct {
	it      task.TaskInterface
	cluster *cluster

	startTime time.Time
	timeout   time.Duration

	onCompleteFn func(id string, val any, err error)
	onFailureFn  func(id string, err error)

	cacheTTL time.Duration

	maxRetries   int
	retryBackoff RetryBackoffStrategy
	jitter       bool
	retryIf      func(err error) bool
	retryMode    RetryMode

	repeatInterval time.Duration
	repeatUntil    time.Time
	repeatCount    int

	priority int
	index    int
}

func (c *clusterTaskBuilder) WithStartTime(st time.Time) ClusterTaskBuilderInterface {
	c.startTime = st
	return c
}

func (c *clusterTaskBuilder) WithTimeout(tm time.Duration) ClusterTaskBuilderInterface {
	c.timeout = tm
	return c
}

func (c *clusterTaskBuilder) WithPriority(p int) ClusterTaskBuilderInterface {
	c.priority = p
	return c
}

func (c *clusterTaskBuilder) WithRetry(retries int) ClusterTaskBuilderInterface {
	c.maxRetries = retries
	return c
}

func (c *clusterTaskBuilder) WithBackoffStrategy(strategy RetryBackoffStrategy) ClusterTaskBuilderInterface {
	c.retryBackoff = strategy
	return c
}

func (c *clusterTaskBuilder) WithJitter() ClusterTaskBuilderInterface {
	c.jitter = true
	return c
}

func (c *clusterTaskBuilder) RetryIf(fn func(err error) bool) ClusterTaskBuilderInterface {
	c.retryIf = fn
	return c
}

func (c *clusterTaskBuilder) WithRetryMode(mode RetryMode) ClusterTaskBuilderInterface {
	c.retryMode = mode
	return c
}

func (c *clusterTaskBuilder) WithRepeatInterval(interval time.Duration) ClusterTaskBuilderInterface {
	c.repeatInterval = interval
	return c
}

func (c *clusterTaskBuilder) RepeatUntil(until time.Time) ClusterTaskBuilderInterface {
	c.repeatUntil = until
	return c
}

func (c *clusterTaskBuilder) WithRepeatCount(n int) ClusterTaskBuilderInterface {
	c.repeatCount = n
	return c
}

func (c *clusterTaskBuilder) WithCache(ttl time.Duration) ClusterTaskBuilderInterface {
	c.cacheTTL = ttl
	return c
}

func (c *clusterTaskBuilder) OnComplete(fn func(id string, val any, err error)) ClusterTaskBuilderInterface {
	c.onCompleteFn = fn
	return c
}

func (c *clusterTaskBuilder) OnFailure(fn func(id string, err error)) ClusterTaskBuilderInterface {
	c.onFailureFn = fn
	return c
}

func (c *clusterTaskBuilder) Submit() (string, error) {
	c.cluster.mu.Lock()
	defer c.cluster.mu.Unlock()

	id := c.it.ID()
	if id == "" {
		id = c.cluster.generateNextID()
		c.it.SetID(id)
	}

	info, exists := c.cluster.subscribers[id]
	if exists {
		if info.ct != nil {
			return "", fmt.Errorf("task with id %s is already running", id)
		}
	} else {
		info = &subscribeInfo{
			waiters: make([]chan Result, 0),
		}
		c.cluster.subscribers[id] = info
	}

	ct := newClusterTask(c.cluster, c.it, c.startTime, c.timeout, c.priority)

	ct.retryCfg = retryConfig{
		maxRetries:   c.maxRetries,
		retryBackoff: c.retryBackoff,
		jitter:       c.jitter,
		retryIf:      c.retryIf,
		retryMode:    c.retryMode,
	}

	ct.onCompleteFn = c.onCompleteFn
	ct.onFailureFn = c.onFailureFn

	ct.repeatCfg = repeatConfig{
		repeatInterval: c.repeatInterval,
		repeatCount:    c.repeatCount,
		repeatUntil:    c.repeatUntil,
	}

	info.ct = ct
	info.cacheTTL = c.cacheTTL
	info.result = nil

	c.cluster.exec.sch.submitInternal(ct)

	return id, nil
}
