package cluster

import (
	"context"
	"fmt"
	"github.com/bozylik/taskara/task"
	"sync"
	"sync/atomic"
	"time"
)

// ClusterInterface - interface that provides encapsulation for cluster methods.
type ClusterInterface interface {
	// Run starts the cluster's execution engine. Workers begin listening to the queue and processing tasks.
	Run()
	// Stop - graceful shutdown. The cluster stops accepting new tasks and waits for active workers to finish.
	// If the timeout is reached before tasks finish, it returns an error.
	Stop(timeout time.Duration) error
	// AddTask - the entry point for submitting a task. It returns a builder that allows you to configure scheduling, timeouts, and metadata.
	// You must call .Submit() at the end of the chain to queue the task.
	AddTask(t task.TaskInterface) ClusterTaskBuilderInterface
	// Subscribe returns a channel that receives the task's result (val and err).
	Subscribe(id string) (<-chan Result, error)
	// Cancel - immediate shutdown. Instantly kills all workers and cancels all active task contexts.
	// Use this only when a graceful shutdown is not possible, as it may leave tasks in an incomplete state.
	Cancel()
	// CancelTask targets and cancels a specific task by its ID.
	// This triggers the cancelled channel inside the TaskFunc and closes the task's context.
	CancelTask(id string)
}

type cluster struct {
	exec *executor

	mu          sync.RWMutex
	subscribers map[string]*subscribeInfo

	ctx    context.Context
	cancel context.CancelFunc

	counter int64
}

// NewCluster is a function (constructor), creates a new cluster instance.
func NewCluster(workers int, ctx context.Context) ClusterInterface {
	if workers <= 0 {
		panic("workers must be greater than 0")
	}

	combinedCtx, cancel := context.WithCancel(ctx)

	c := &cluster{
		subscribers: make(map[string]*subscribeInfo),
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

func (c *cluster) AddTask(t task.TaskInterface) ClusterTaskBuilderInterface {
	if t == nil {
		panic("task cannot be nil")
	}

	return &clusterTaskBuilder{
		it:           t,
		cluster:      c,
		retryMode:    Requeue,
		retryBackoff: FixedBackoff{Delay: 1 * time.Second},
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

	if err != nil && info.ct.retryMode == Requeue && info.ct.shouldRetry(err) {
		delay := info.ct.calculateNextDelay()
		info.ct.prepareForRetry()
		info.ct.startTime = time.Now().Add(delay)
		c.exec.sch.submitInternal(info.ct)

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

	info.ct = nil

	if info.isCacheable {
		oldInfo := info
		time.AfterFunc(5*time.Minute, func() {
			c.mu.Lock()
			defer c.mu.Unlock()
			if cur, exists := c.subscribers[id]; exists && cur == oldInfo {
				delete(c.subscribers, id)
			}
		})
		c.mu.Unlock()
	} else {
		delete(c.subscribers, id)
		c.mu.Unlock()
	}
}

func (c *cluster) CancelTask(id string) {
	c.mu.Lock()
	info, ok := c.subscribers[id]
	c.mu.Unlock()

	if ok && info.ct != nil {
		info.ct.cancelClusterTask()
	}
}
