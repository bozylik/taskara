package cluster

import (
	"context"
	"errors"
	"fmt"
	"github.com/bozylik/taskara/task"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"
)

// Result is what you get from the subscription channel.
type Result struct {
	// Data from the task.
	Result any
	
	// Error from the task, timeout, or panic.
	Err error
}

type clusterTask struct {
	task    task.TaskInterface
	cluster *cluster
	status  atomic.Int32

	ctx    context.Context
	cancel context.CancelFunc

	done chan struct{}
	once sync.Once

	onCompleteFn func(id string, val any, err error)
	onFailureFn  func(id string, err error)

	startTime time.Time
	timeout   time.Duration

	retryCfg retryConfig

	Priority int
	Index    int
}

func newClusterTask(c *cluster, t task.TaskInterface, st time.Time, to time.Duration, p int) *clusterTask {
	ctx, cancel := context.WithCancel(c.ctx)
	return &clusterTask{
		task:      t,
		cluster:   c,
		ctx:       ctx,
		cancel:    cancel,
		done:      make(chan struct{}),
		startTime: st,
		timeout:   to,
		Priority:  p,
		once:      sync.Once{},
	}
}

func (c *clusterTask) getID() string {
	return c.task.ID()
}

func (c *clusterTask) getStatus() task.TaskStatus {
	return task.TaskStatus(c.status.Load())
}

func (c *clusterTask) cancelClusterTask() {
	if c.getStatus() < task.StatusCompleted {
		c.cancel()
	}
}

func (c *clusterTask) shouldRetry(err error) bool {
	if c.retryCfg.maxRetries <= 0 || c.retryCfg.currentAttempt >= c.retryCfg.maxRetries {
		return false
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	if c.retryCfg.retryIf != nil {
		return c.retryCfg.retryIf(err)
	}

	return true
}

func (c *clusterTask) calculateNextDelay() time.Duration {
	if c.retryCfg.retryBackoff == nil {
		return 0
	}

	delay := c.retryCfg.retryBackoff.Next(c.retryCfg.currentAttempt)
	if c.retryCfg.jitter {
		f := 0.9 + rand.Float64()*0.2
		delay = time.Duration(float64(delay) * f)
	}
	return delay
}

func (c *clusterTask) prepareForRetry() {
	c.status.Store(int32(task.StatusWaiting))
	c.ctx, c.cancel = context.WithCancel(c.cluster.ctx)
	c.retryCfg.currentAttempt++
}

func (c *clusterTask) runClusterTask(workerCtx context.Context, report task.Reporter) {
	taskID := c.getID()

	if c.ctx.Err() != nil {
		c.finish(task.StatusCancelled, c.ctx.Err(), report, nil)
		return
	}

	var once sync.Once
	finalReport := func(status task.TaskStatus, err error, val any) {
		once.Do(func() {
			c.finish(status, err, report, val)
		})
	}

	handleContextDone := func() {
		if c.ctx.Err() != nil {
			finalReport(task.StatusCancelled, c.ctx.Err(), nil)
		} else {
			finalReport(task.StatusCancelled, workerCtx.Err(), nil)
		}
	}

	defer func() {
		if r := recover(); r != nil {
			finalReport(task.StatusError, fmt.Errorf("internal cluster panic: %v", r), nil)
		}
	}()

	for {
		c.status.Store(int32(task.StatusRunning))

		type attemptResult struct {
			val      any
			err      error
			panicked bool
		}

		resCh := make(chan attemptResult, 1)
		fnExited := make(chan struct{})

		go func() {
			defer func() {
				if r := recover(); r != nil {
					resCh <- attemptResult{err: fmt.Errorf("task panicked: %v", r), panicked: true}
				}
				close(fnExited)
			}()

			c.task.Fn()(taskID, workerCtx, c.ctx.Done(), func(id string, val any, err error) {
				select {
				case resCh <- attemptResult{val: val, err: err}:
				default:
				}
			})
		}()

		var currentRes attemptResult
		var hasResult bool

		select {
		case currentRes = <-resCh:
			hasResult = true

		case <-fnExited:
			if c.ctx.Err() != nil || workerCtx.Err() != nil {
				handleContextDone()
				return
			}
			if !hasResult {
				currentRes = attemptResult{val: nil, err: nil}
			}

		case <-workerCtx.Done():
			handleContextDone()
			return
		}

		if currentRes.err == nil {
			finalReport(task.StatusCompleted, nil, currentRes.val)
			return
		}

		if c.retryCfg.retryMode == Immediate && c.shouldRetry(currentRes.err) {
			c.prepareForRetry()
			delay := c.calculateNextDelay()

			timer := time.NewTimer(delay)

			select {
			case <-timer.C:
				continue

			case <-workerCtx.Done():
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}

				if c.ctx.Err() != nil {
					finalReport(task.StatusCancelled, c.ctx.Err(), nil)
				} else {
					finalReport(task.StatusCancelled, workerCtx.Err(), nil)
				}
				return
			}
		}

		if c.retryCfg.retryMode == Requeue && c.shouldRetry(currentRes.err) {
			if report != nil {
				report(c.getID(), currentRes.val, currentRes.err)
			}
			return
		}

		status := task.StatusError
		finalReport(status, currentRes.err, currentRes.val)
		return
	}
}

func (c *clusterTask) finish(status task.TaskStatus, err error, report task.Reporter, val any) {
	c.once.Do(func() {
		c.status.Store(int32(status))
		c.cancel()

		if err != nil && c.onFailureFn != nil {
			c.onFailureFn(c.getID(), err)
		}

		if c.onCompleteFn != nil {
			c.onCompleteFn(c.getID(), val, err)
		}

		if report != nil {
			report(c.getID(), val, err)
		}

		close(c.done)
	})
}
