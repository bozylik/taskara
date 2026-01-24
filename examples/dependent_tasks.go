package main

import (
	"context"
	"fmt"
	"github.com/bozylik/taskara/cluster"
	"github.com/bozylik/taskara/task"
	"time"
)

// Version 0.0.2-alpha does not represent dependent tasks directly,
// you can subscribe to tasks using the Subscribe method
// but if all tasks with the current number of workers are waiting for results, a Deadlock will occur.
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	myCluster := cluster.NewCluster(2, ctx)
	myCluster.Run()

	job1 := func(id string, ctx context.Context, cancelled <-chan struct{}, report task.Reporter) {
		fmt.Printf("[%s] Working job1...\n", id)

		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
			fmt.Printf("[%s] Waiting 5 seconds...\n", id)
			report(id, "Data-from-Task-1", nil)
		}
	}

	task1 := task.NewTask("", job1)

	clusterTaskID, err := myCluster.AddTask(task1).Submit()
	if err != nil {
		// Panic for example
		panic(err)
	}

	job2 := func(id string, ctx context.Context, cancelled <-chan struct{}, report task.Reporter) {
		fmt.Printf("[%s] Working job2...\n", id)

		// Using closure with clusterTaskID
		results, err := myCluster.Subscribe(clusterTaskID)
		if results == nil || err != nil {
			fmt.Printf("[%s] error: %s\n", id, err)
		}

		fmt.Printf("[%s] Got results from Task1: %s", id, <-results)
		report(id, "Data-from-Task2", nil)
	}

	task2 := task.NewTask("task-2", job2)
	_, err = myCluster.AddTask(task2).Submit()
	if err != nil {
		// Panic for example
		panic(err)
	}

	// Just wait result
	time.Sleep(10 * time.Second)
}
