# taskara

### lightweight task scheduler with priority and worker pool

taskara is a simple and fast library for managing tasks in go. it allows you to run many tasks concurrently using a controlled number of workers. it is designed to be minimal, having zero external dependencies.

---

> [!IMPORTANT]
> ## alpha-build
> this is an early alpha version of the project. it does not include detailed instructions or many examples yet.<br>
> **status:** under active development.<br>
>**future:** we plan to add more features, better documentation, and complex examples soon.<br>
>we do not guarantee backward compatibility of various releases until the first stable version is available.

---

## navigation
* [features](#features)
* [architecture](#architecture)
* [how to use](#how-to-use)
* [examples](#examples)

---

## features

* **worker pool:** limit the number of active goroutines to save resources.
* **priority queue:** important tasks are executed first.
* **scheduling:** start tasks immediately or at a specific time.
* **timeouts:** automatically cancel tasks that take too long.
* **individual control:** cancel a specific task by its id without stopping others.
* **easy integration:** get results through go channels.

---

## architecture

the system consists of three main parts:
1. **scheduler:** manages the waiting and ready queues using a priority heap.
2. **executor:** runs a fixed pool of workers that process tasks.
3. **cluster:** provides a simple interface to submit and manage tasks.

---

## how to use

>**installation**<br>

`go get github.com/bozylik/taskara@v0.0.2-alpha`

Usage:
```go
import (
    "github.com/bozylik/taskara/task"
    "github.com/bozylik/taskara/cluster"
)
```

>**part 1: TaskFunc (job)**<br>

**TaskFunc** - is a type from the `taskara/task` package that defines the function (job) to be executed:<br>
  `type TaskFunc func(id string, workerCtx context.Context, cancelled <-chan struct{}, report Reporter)`
  ```go
  job1 := func(id string, ctx context.Context, cancelled <-chan struct{}, report task.Reporter) {

		select {
		// Specific case: handle manual cancellation explicitly if needed
		case <-cancelled:
			return
		// General case: triggers on manual cancel, timeout, and cluster shutdown
		case <-ctx.Done():
			fmt.Println("timeout")
			return
		case <-time.After(4 * time.Second):
			fmt.Printf("[%s] Working job1...\n", id)
		}

		// Report results to the cluster
		report(id, "Data from task-1", nil)
	}
  ```
> [!NOTE] 
>**Semantic of Cancellation:** ctx.Done() is a universal signal. It closes whether the task timed out, the cluster was stopped, or CancelTask() was called. Use the cancelled channel only when you need to distinguish a manual user action from a system timeout.<br>
>
>**Automatic Cleanup:** If your function returns (via return) without calling report(), the cluster's internal defer mechanism will automatically finalize the task and set its status to Cancelled or Completed with nil values.


>**part 2: task and TaskInterface**<br>

A Task is a base struct used within clusters. It contains a unique ID and the function to be executed.<br>
**task** - struct with `id string` and `fn TaskFunc` fields.<br>
**TaskInterface** - interface that provides encapsulation for task methods.<br>
```go
// func NewTask(id string, fn TaskFunc) TaskInterface
// The task ID will be set automatically by the cluster if you leave it empty
task1 := task.NewTask("", job1)
// or specify a custom ID
task1 := task.NewTask("task-1", job1)
```

>**part 3: cluster and ClusterInterface**<br>

The Cluster manages task execution. It contains all necessary methods for running, cancelling tasks, and the cluster itself.<br>
**cluster** - struct that contains executor, scheduler, and state.<br>
**ClusterInterface** - interface that provides encapsulation for cluster methods.<br>
```go
// Cluster context
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// func NewCluster(workers int, ctx context.Context) ClusterInterface
// Create cluster with 2 active workers
myCluster := cluster.NewCluster(2, ctx)
// Run the cluster now or later (after adding tasks) (it will start processing the queue)
myCluster.Run()

// Management:
// myCluster.Stop(timeout) // graceful shutdown
// myCluster.Cancel()      // immediate cancellation
```
>**part 4: clusterTask and clusterTaskBuilder**<br>

**clusterTask** - is an internal task struct that represents a task within the cluster's lifecycle.<br>
**clusterTaskBuilder** - is a helper that allows you to use chaining while adding a new task.<br>
```go
// Add task to the cluster with optional parameters
clusterTaskID, err := myCluster.AddTask(task1).
		WithStartTime(time.Now().Add(5 * time.Second)).
		WithTimeout(5 * time.Second).
    IsCacheable(true).     // Cache the result for later retrieval
		Submit()           // Returns the cluster-generated task ID 

if err != nil {
		// Panic for example
		panic(err)
	}

// Subscribe to get the result
// Note: You should subscribe before completion unless IsCacheable(true) is used
resChan, err := myCluster.Subscribe(clusterTaskID)
if err != nil {
  // Panic for example
  panic(err)
}

fmt.Println("Results:", <-resChan)
```

---

## examples

Check out our [usage examples](https://github.com/bozylik/taskara/tree/main/examples), to see more complex use cases and implementation details.
