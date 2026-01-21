package main

import (
	"context"
	"fmt"
	"github.com/bozylik/taskara/cluster"
	"github.com/bozylik/taskara/task"
	"time"
)

func main() {
	// 1. create cluster with 2 workers
	ctx := context.Background()
	mycluster := cluster.NewCluster(2, ctx)
	mycluster.Run()

	// 2. define a long task
	heavyjob := func(workerctx context.Context, cancelled <-chan struct{}, report task.Reporter) {
		fmt.Println("[task] starting long work...")

		for i := 1; i <= 10; i++ {
			select {
			case <-workerctx.Done():
				fmt.Println("[task] cluster (worker) context closed...")
				return
			case <-cancelled:
				// cluster asked to stop this specific task
				fmt.Println("[task] cancel signal received. cleaning up...")
				return
			case <-time.After(500 * time.Millisecond):
				// simulate work
				fmt.Printf("[task] step %d/10 finished\n", i)
			}
		}

		// if finished, report success
		report("task-1", "success!", nil)
	}

	// 3. create task and add to cluster
	t := task.NewTask("task-1", heavyjob)

	// subscribe to results
	reschan, _ := mycluster.Subscribe("task-1")

	mycluster.AddTask(t).
		WithPriority(10).
		WithTimeout(10 * time.Second).
		Submit()

	// 4. wait 2 seconds then cancel
	time.Sleep(2 * time.Second)
	fmt.Println("[main] deciding to cancel the task!")

	// cancel everything in the cluster
	mycluster.Cancel()

	// 5. wait for the result
	result := <-reschan
	if result.Err != nil {
		fmt.Printf("[main] task finished with error/cancel: %v\n", result.Err)
	} else {
		fmt.Printf("[main] task finished successfully: %v\n", result.Result)
	}

	mycluster.Stop(1 * time.Second)
}
