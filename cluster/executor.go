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

				workerCtx := ctx

				if taskItem.timeout > 0 {
					var cancel context.CancelFunc
					workerCtx, cancel = context.WithTimeout(ctx, taskItem.timeout)

					taskItem.Run(workerCtx, e.reporter)
					cancel()
				} else {
					taskItem.Run(workerCtx, e.reporter)
				}
			}
		}()
	}
}
