package cluster

import (
	"context"
	"fmt"
	"github.com/bozylik/taskara/task"
	"sync"
	"time"
)

type ClusterInterface interface {
	Run()
	Stop(timeout time.Duration) error
	AddTask(t task.TaskInterface) *clusterTaskBuilder
	Subscribe(id string) (<-chan Result, error)
	Cancel()
	CancelTask(id string)
}

type cluster struct {
	exec *executor

	mu          sync.RWMutex
	subscribers map[string]*SubscribeInfo

	ctx    context.Context
	cancel context.CancelFunc
}

func NewCluster(workers int, ctx context.Context) ClusterInterface {
	if workers <= 0 {
		panic("workers must greater than 0")
	}

	combinedCtx, cancel := context.WithCancel(ctx)

	c := &cluster{
		subscribers: make(map[string]*SubscribeInfo),
		ctx:         combinedCtx,
		cancel:      cancel,
	}

	c.exec = newExecutor(workers, c.SetResult)

	return c
}

func (c *cluster) Cancel() {
	c.cancel()

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, info := range c.subscribers {
		for _, ch := range info.waiters {
			ch <- Result{Err: c.ctx.Err()}
			close(ch)
		}

		info.waiters = nil
		if info.err == nil {
			info.err = c.ctx.Err()
		}
	}
}

func (e *executor) wait() {
	e.wg.Wait()
}

func (c *cluster) Stop(timeout time.Duration) error {
	c.cancel()

	done := make(chan struct{})
	go func() {
		c.exec.wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timeout: some tasks may still be running")
	}
}

func (c *cluster) AddTask(t task.TaskInterface) *clusterTaskBuilder {
	if t == nil {
		panic("task cannot be nil")
	}

	return &clusterTaskBuilder{
		it:      t,
		cluster: c,
	}
}

func (c *cluster) Run() {
	c.exec.runExecutor(c.ctx)
}

func (c *cluster) Subscribe(id string) (<-chan Result, error) {
	select {
	case <-c.ctx.Done():
		return nil, fmt.Errorf("cluster is closed: %w", c.ctx.Err())
	default:
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	info, ok := c.subscribers[id]

	if ok && (info.result != nil || info.err != nil) {
		return nil, fmt.Errorf("task %s already finished, result is no longer available", id)
	}

	if !ok {
		info = &SubscribeInfo{
			waiters: make([]chan Result, 0),
		}
		c.subscribers[id] = info
	}

	ch := make(chan Result, 1)
	info.waiters = append(info.waiters, ch)
	info.subs++

	return ch, nil
}

func (c *cluster) SetResult(id string, val any, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	info, ok := c.subscribers[id]
	if !ok {
		return
	}

	res := Result{Result: val, Err: err}
	for _, ch := range info.waiters {
		ch <- res
		close(ch)
	}

	delete(c.subscribers, id)
}

func (c *cluster) CancelTask(id string) {
	c.mu.Lock()
	info, ok := c.subscribers[id]
	c.mu.Unlock()

	if ok && info.ct != nil {
		info.ct.Cancel()
	}
}
