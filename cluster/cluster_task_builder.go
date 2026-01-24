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

func (c *clusterTaskBuilder) WithStartTime(st time.Time) *clusterTaskBuilder {
	c.startTime = st
	return c
}

func (c *clusterTaskBuilder) WithTimeout(tm time.Duration) *clusterTaskBuilder {
	c.timeout = tm
	return c
}

func (c *clusterTaskBuilder) WithPriority(p int) *clusterTaskBuilder {
	c.priority = p
	return c
}

func (c *clusterTaskBuilder) IsCacheable(v bool) *clusterTaskBuilder {
	c.isCacheable = v
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

	if info, exists := c.cluster.subscribers[id]; exists && info.ct != nil {
		return "", fmt.Errorf("task with id %s is already running", id)
	}

	ct := newClusterTask(c.cluster.ctx, c.it, c.startTime, c.timeout, c.priority)

	info, ok := c.cluster.subscribers[id]
	if !ok {
		info = &SubscribeInfo{waiters: make([]chan Result, 0)}
		c.cluster.subscribers[id] = info
	}

	info.ct = ct
	info.isCacheable = c.isCacheable

	info.result = nil

	c.cluster.exec.sch.submitInternal(ct)

	return id, nil
}
