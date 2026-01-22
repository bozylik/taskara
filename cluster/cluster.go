package cluster

import (
	"context"
	"fmt"
	"github.com/bozylik/taskara/task"
	"sync"
	"sync/atomic"
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

	counter int64
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

	c.exec = newExecutor(workers, c.setResult)

	return c
}

func (c *cluster) generateNextID() string {
	id := atomic.AddInt64(&c.counter, 1)
	return fmt.Sprintf("task-%d", id)
}

func (c *cluster) Cancel() {
	c.cancel()

	c.mu.Lock()
	defer c.mu.Unlock()

	for id, info := range c.subscribers {
		cancelRes := Result{Err: c.ctx.Err()}

		for _, ch := range info.waiters {
			ch <- cancelRes
			close(ch)
		}
		info.waiters = nil

		if info.result == nil {
			info.result = &cancelRes
		}

		delete(c.subscribers, id)
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

	if ok && info.result != nil {
		ch := make(chan Result, 1)
		ch <- *info.result
		close(ch)
		return ch, nil
	}

	if !ok {
		return nil, fmt.Errorf("task %s not found: it was never submitted or cache expired", id)
	}

	ch := make(chan Result, 1)
	info.waiters = append(info.waiters, ch)
	info.subs++

	return ch, nil
}

func (c *cluster) setResult(id string, val any, err error) {
	c.mu.Lock()
	info, ok := c.subscribers[id]
	if !ok {
		c.mu.Unlock()
		return
	}

	res := Result{Result: val, Err: err}
	info.result = &res

	for _, ch := range info.waiters {
		ch <- res
		close(ch)
	}
	info.waiters = nil
	c.mu.Unlock()

	if info.isCacheable {
		time.AfterFunc(5*time.Minute, func() {
			c.mu.Lock()
			delete(c.subscribers, id)
			c.mu.Unlock()
			fmt.Printf("Cache for task %s cleared\n", id)
		})
	} else {
		c.mu.Lock()
		delete(c.subscribers, id)
		c.mu.Unlock()
	}
}

func (c *cluster) CancelTask(id string) {
	c.mu.Lock()
	info, ok := c.subscribers[id]
	c.mu.Unlock()

	if ok && info.ct != nil {
		info.ct.Cancel()
	}
}
