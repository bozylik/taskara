package main

import (
	"context"
	"fmt"
	"github.com/bozylik/taskara/cluster"
	"github.com/bozylik/taskara/task"
	"time"
)

func main() {
	myCluster := cluster.NewCluster(1, context.Background())
	myCluster.Run()

	job := func(id string, ctx context.Context, cancel <-chan struct{}, report task.Reporter) {
		time.Sleep(10 * time.Second)
		report(id, nil, fmt.Errorf("error"))
	}

	_, _ = myCluster.AddTask(task.NewTask("task-1", job)).
		WithRetry(1).
		WithRetryMode(cluster.Immediate).
		WithBackoffStrategy(cluster.FixedBackoff{Delay: 10 * time.Second}).
		Submit()

	time.Sleep(1000 * time.Millisecond)
	myCluster.CancelTask("task-1")
	time.Sleep(500 * time.Millisecond)

}
