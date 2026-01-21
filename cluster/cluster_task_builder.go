package cluster

import (
	"github.com/bozylik/taskara/task"
	"time"
)

type clusterTaskBuilder struct {
	it      task.TaskInterface
	cluster *cluster

	startTime time.Time
	timeout   time.Duration

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

func (c *clusterTaskBuilder) Submit() ClusterInterface {
	ct := newClusterTask(c.it, c.startTime, c.timeout, c.priority)

	c.cluster.mu.Lock()
	info, ok := c.cluster.subscribers[ct.ID()]
	if !ok {
		info = &SubscribeInfo{waiters: make([]chan Result, 0)}
		c.cluster.subscribers[ct.ID()] = info
	}

	info.ct = ct
	c.cluster.mu.Unlock()

	c.cluster.exec.sch.submitInternal(ct)
	return c.cluster
}
