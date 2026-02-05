package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/bozylik/taskara/cluster"
	"github.com/bozylik/taskara/task"
	"time"
)

func main() {
	myCluster := cluster.NewCluster(1, context.Background())
	myCluster.Run()

	jobError := fmt.Errorf("job error")
	job := func(id string, ctx context.Context, cancel <-chan struct{}, report task.Reporter) {
		fmt.Println(time.Now(), "- job started")
		report(id, nil, jobError)
	}

	taskID, err := myCluster.AddTask(task.NewTask("task-1", job)).
		WithRetry(3).
		WithRetryMode(cluster.Immediate).
		WithBackoffStrategy(cluster.NewFixedBackoff(1 * time.Second)).
		RetryIf(func(err error) bool {
			if errors.Is(err, jobError) {
				return true
			}

			return false
		}).
		OnComplete(func(id string, val any, err error) {
			fmt.Println("OnComplete:", id)
		}).
		OnFailure(func(id string, err error) {
			fmt.Println("OnFailure:", id)
		}).
		Submit()

	if err != nil {
		// panic for example
		panic(err)
	}

	res, err := myCluster.Subscribe(taskID)
	if res == nil || err != nil {
		// panic for example
		panic(res)
	}

	fmt.Println(<-res)
}
