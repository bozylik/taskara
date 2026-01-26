package cluster

import (
	"context"
	"sync"
)

type executor struct {
	sch         *scheduler
	workerCount int
	reporter    func(string, any, error)
	wg          sync.WaitGroup
}

func newExecutor(workers int, reporter func(string, any, error)) *executor {
	return &executor{
		sch:         newScheduler(),
		workerCount: workers,
		reporter:    reporter,
	}
}

func (e *executor) runExecutor(ctx context.Context) {
	go e.sch.runScheduler(ctx)

	for i := 0; i < e.workerCount; i++ {
		e.wg.Add(1)
		go func() {
			defer e.wg.Done()
			for {
				taskItem := e.sch.getNext(ctx)
				if taskItem == nil {
					return
				}

				workerCtx := taskItem.ctx
				var cancel context.CancelFunc

				if taskItem.timeout > 0 {
					workerCtx, cancel = context.WithTimeout(taskItem.ctx, taskItem.timeout)
				}

				taskItem.runClusterTask(workerCtx, e.reporter)

				if cancel != nil {
					cancel()
				}
			}
		}()
	}
}
