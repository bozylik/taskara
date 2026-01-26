package cluster

import (
	"fmt"
	"github.com/bozylik/taskara/task"
	"time"
)

type clusterTaskBuilder struct {
	it      task.TaskInterface
	cluster *cluster

	startTime time.Time
	timeout   time.Duration

	isCacheable bool

	priority int
	index    int
}

// Schedules the task to run at a specific time.
// If the time is in the past or time.Now(), the task will be executed as soon as a worker is available.
func (c *clusterTaskBuilder) WithStartTime(st time.Time) *clusterTaskBuilder {
	c.startTime = st
	return c
}

// Sets a maximum execution time for the task.
// If the task exceeds this duration, its ctx will be cancelled, and the task will be marked as timed out.
func (c *clusterTaskBuilder) WithTimeout(tm time.Duration) *clusterTaskBuilder {
	c.timeout = tm
	return c
}

// Sets the task's priority.
// Higher values (or lower, depending on your heap logic—usually higher) will move the task to the front of the queue.
func (c *clusterTaskBuilder) WithPriority(p int) *clusterTaskBuilder {
	c.priority = p
	return c
}

// Determines if the task result should be stored in memory after completion.
// Currently, cached results are stored for 5 minutes after the task completes.
// After this period, the result is purged from memory to prevent leaks.
// (Note: This duration may become configurable in future releases).
func (c *clusterTaskBuilder) IsCacheable(v bool) *clusterTaskBuilder {
	c.isCacheable = v
	return c
}

// The final method in the chain.
// It validates the task, generates an ID (if empty), and pushes the task into the scheduler.
// Returns an error if a task with the same ID is already running or managed by the cluster.
func (c *clusterTaskBuilder) Submit() (string, error) {
	c.cluster.mu.Lock()
	defer c.cluster.mu.Unlock()

	id := c.it.ID()

	if id == "" {
		id = c.cluster.generateNextID()
		c.it.SetID(id)
	}

	if info, exists := c.cluster.subscribers[id]; exists && info.ct != nil {
		return "", fmt.Errorf("task with id %s is already running", id)
	}

	ct := newClusterTask(c.cluster.ctx, c.it, c.startTime, c.timeout, c.priority)

	info, ok := c.cluster.subscribers[id]
	if !ok {
		info = &subscribeInfo{waiters: make([]chan Result, 0)}
		c.cluster.subscribers[id] = info
	}

	info.ct = ct
	info.isCacheable = c.isCacheable

	info.result = nil

	c.cluster.exec.sch.submitInternal(ct)

	return id, nil
}
