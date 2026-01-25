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

func newClusterTask(clusterCtx context.Context, t task.TaskInterface, st time.Time, to time.Duration, p int) *clusterTask {
	ctx, cancel := context.WithCancel(clusterCtx)
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
	taskID := c.ID()

	if c.ctx.Err() != nil {
		c.finish(task.StatusCancelled, c.ctx.Err(), report, nil)
		return
	}

	c.status.Store(int32(task.StatusRunning))

	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("task panicked: %v", r)
			c.finish(task.StatusError, err, report, nil)
		} else {
			select {
			case <-c.ctx.Done():
				c.finish(task.StatusCancelled, c.ctx.Err(), report, nil)
			case <-workerCtx.Done():
				c.finish(task.StatusCancelled, workerCtx.Err(), report, nil)
			default:
				c.finish(task.StatusCompleted, nil, report, nil)
			}
		}
	}()

	taskLogic := c.task.Fn()

	taskLogic(taskID, workerCtx, c.ctx.Done(), func(id string, val any, err error) {
		finalStatus := task.StatusCompleted
		finalErr := err

		select {
		case <-c.ctx.Done():
			finalStatus = task.StatusCancelled
			finalErr = c.ctx.Err()
			val = nil
		case <-workerCtx.Done():
			finalStatus = task.StatusCancelled
			finalErr = workerCtx.Err()
			val = nil
		default:
			if err != nil {
				finalStatus = task.StatusError
			}
		}

		c.finish(finalStatus, finalErr, report, val)
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
