package cluster

import (
	"context"
	"fmt"
	"github.com/bozylik/taskara/task"
	"sync"
	"sync/atomic"
	"time"
)

type Result struct {
	Result any
	Err    error
}

type InternalTaskInterface interface {
	ID() string
	Run(ctx context.Context, report task.Reporter)
	Status() task.TaskStatus
	Cancel()
}

type clusterTask struct {
	task   task.TaskInterface
	status atomic.Int32

	ctx    context.Context
	cancel context.CancelFunc

	done chan struct{}
	once sync.Once

	startTime time.Time
	timeout   time.Duration

	Priority int
	Index    int
}

func newClusterTask(t task.TaskInterface, st time.Time, to time.Duration, p int) *clusterTask {
	ctx, cancel := context.WithCancel(context.Background())
	return &clusterTask{
		task:      t,
		ctx:       ctx,
		cancel:    cancel,
		done:      make(chan struct{}),
		startTime: st,
		timeout:   to,
		Priority:  p,
	}
}

func (c *clusterTask) ID() string {
	return c.task.ID()
}

func (c *clusterTask) Status() task.TaskStatus {
	return task.TaskStatus(c.status.Load())
}

func (c *clusterTask) Cancel() {
	if c.Status() < task.StatusCompleted {
		c.cancel()
	}
}

func (c *clusterTask) Run(workerCtx context.Context, report task.Reporter) {
	if c.ctx.Err() != nil {
		c.finish(task.StatusCancelled, c.ctx.Err(), report, nil)
		return
	}

	c.status.Store(int32(task.StatusRunning))

	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("task panicked: %v", r)
			c.finish(task.StatusError, err, report, nil)
		} else if c.Status() == task.StatusRunning {
			if c.ctx.Err() != nil || workerCtx.Err() != nil {
				c.finish(task.StatusCancelled, context.Cause(c.ctx), report, nil)
			} else {
				c.finish(task.StatusCompleted, nil, report, nil)
			}
		}
	}()

	taskLogic := c.task.Fn()
	taskLogic(workerCtx, c.ctx.Done(), func(id string, val any, err error) {
		status := task.StatusCompleted
		select {
		case <-c.ctx.Done():
			status = task.StatusCancelled
			err = c.ctx.Err()
		case <-workerCtx.Done():
			status = task.StatusCancelled
			err = workerCtx.Err()
		default:
			if err != nil {
				status = task.StatusError
			}
		}
		c.finish(status, err, report, val)
	})
}

func (c *clusterTask) finish(status task.TaskStatus, err error, report task.Reporter, val any) {
	c.once.Do(func() {
		c.status.Store(int32(status))
		c.cancel()

		if report != nil {
			report(c.ID(), val, err)
		}
		close(c.done)
	})
}
