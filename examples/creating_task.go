package main

import (
	"context"
	"fmt"
	"github.com/bozylik/taskara/cluster"
	"github.com/bozylik/taskara/task"
	"time"
)

// The task acts as an independent unit, but it is impossible to imagine it without a cluster.
func main() {
	// Cluster context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Creating cluster
	myCluster := cluster.NewCluster(4, ctx)
	// You can run cluster now or later
	myCluster.Run()

	// Task function definition
	job1 := func(id string, ctx context.Context, cancelled <-chan struct{}, report task.Reporter) {

		select {
		// Manual task cancellation (cluster.CancelTask())
		case <-cancelled:
			return
		// Manual task cancellation, cluster cancellation (cluster cancel) or timeout (WithTimeout())
		case <-ctx.Done():
			fmt.Println("timeout")
			// Automatically report with nil value
			return
		case <-time.After(4 * time.Second):
			fmt.Printf("[%s] Working job1...\n", id)
		}

		// You should call report at the end of the function; if you forget to do it, it will be automatically set to nil.
		report(id, "Data from task-1", nil)
	}

	// The task getID will be set automatically by the cluster if you leave it empty
	task1 := task.NewTask("", job1)
	// Adding task to cluster and getting cluster task getID
	// Using chaining for adding cluster task parameters
	clusterTaskID, err := myCluster.AddTask(task1).
		WithStartTime(time.Now().Add(5 * time.Second)).
		WithTimeout(5 * time.Second).
		IsCacheable(true).
		Submit()

	if err != nil {
		// Panic for example
		panic(err)
	}

	// You must subscribe to the task before it returns a result, otherwise you may get an error.
	// If you need subscriptions after completing the task,
	// the IsCacheable parameter is used in chaining, then the result will be saved for 5 minutes.
	resChan, err := myCluster.Subscribe(clusterTaskID)
	if err != nil {
		// Panic for example
		panic(err)
	}

	// You can cancel task or full cluster
	// myCluster.CancelTask(clusterTaskID)
	// myCluster.Cancel()
	// myCluster.Stop(timeout time.Duration)

	fmt.Println("Results from task-1:", <-resChan)
	time.Sleep(15 * time.Second)
}
