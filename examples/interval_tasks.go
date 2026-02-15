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

	jobError := fmt.Errorf("job error")
	job := func(id string, ctx context.Context, cancel <-chan struct{}, report task.Reporter) {
		//time.Sleep(5 * time.Second)
		fmt.Println(time.Now(), "- job started")
		report(id, nil, jobError)
	}

	taskID, err := myCluster.AddTask(task.NewTask("task-1", job)).
		WithRepeatInterval(10 * time.Second).
		RepeatUntil(time.Now().Add(2 * time.Second)).
		//WithRepeatCount(1).
		Submit()

	if err != nil {
		panic(err)
	}

	res, err := myCluster.Subscribe(taskID)
	if res == nil || err != nil {
		// panic for example
		panic(res)
	}

	fmt.Println(<-res)

	time.Sleep(105 * time.Second)
}
