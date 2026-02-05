package cluster_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/bozylik/taskara/cluster"
	"github.com/bozylik/taskara/task"
	"testing"
	"time"
)

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || (len(s) > len(substr) && s[1:] == substr)
}

func TestCluster_FullIntegrationFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	myCluster := cluster.NewCluster(2, ctx)
	myCluster.Run()

	job1 := func(id string, ctx context.Context, cancelled <-chan struct{}, report task.Reporter) {
		select {
		case <-cancelled:
			return
		case <-ctx.Done():
			fmt.Println("Task context cancelled or timed out")
			return
		case <-time.After(1 * time.Second):
			report(id, "Data from task-1", nil)
		}
	}

	task1 := task.NewTask("integration-task-1", job1)

	startTime := time.Now().Add(2 * time.Second)

	clusterTaskID, err := myCluster.AddTask(task1).
		WithStartTime(startTime).
		WithTimeout(5 * time.Second).
		OnComplete(func(id string, val any, err error) {

		}).
		OnFailure(func(id string, err error) {
			fmt.Println("OnFailure task-1")
		}).
		WithPriority(10).
		Submit()

	if err != nil {
		t.Fatalf("Failed to submit task: %v", err)
	}

	resChan, err := myCluster.Subscribe(clusterTaskID)
	if err != nil {
		t.Fatalf("Failed to subscribe to task: %v", err)
	}

	select {
	case res := <-resChan:
		if time.Now().Before(startTime) {
			t.Errorf("Task finished too early: at %v, but startTime was %v", time.Now(), startTime)
		}

		if res.Err != nil {
			t.Errorf("Task returned error: %v", res.Err)
		}

		if res.Result != "Data from task-1" {
			t.Errorf("Expected 'Data from task-1', got '%v'", res.Result)
		}

	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for task result")
	}

	time.Sleep(500 * time.Millisecond)
}

func TestCluster_CancelTask(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	myCluster := cluster.NewCluster(2, ctx)
	myCluster.Run()

	job1 := func(id string, ctx context.Context, cancelled <-chan struct{}, report task.Reporter) {
		report(id, "Data from task-1", nil)
	}

	task1 := task.NewTask("", job1)

	clusterTaskID, err := myCluster.AddTask(task1).
		Submit()

	if err != nil {
		t.Fatalf("Failed to submit task: %v", err)
	}

	myCluster.CancelTask(clusterTaskID)
}

func TestCluster_CancelCluster(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	myCluster := cluster.NewCluster(2, ctx)
	myCluster.Run()

	job1 := func(id string, ctx context.Context, cancelled <-chan struct{}, report task.Reporter) {
		report(id, "Data from task-1", nil)
	}

	task1 := task.NewTask("", job1)

	_, err := myCluster.AddTask(task1).
		Submit()

	if err != nil {
		t.Fatalf("Failed to submit task: %v", err)
	}

	myCluster.Cancel()
}

func TestCluster_StopCluster(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	myCluster := cluster.NewCluster(2, ctx)
	myCluster.Run()

	job1 := func(id string, ctx context.Context, cancelled <-chan struct{}, report task.Reporter) {
		report(id, "Data from task-1", nil)
	}

	task1 := task.NewTask("", job1)

	_, err := myCluster.AddTask(task1).
		Submit()

	if err != nil {
		t.Fatalf("Failed to submit task: %v", err)
	}

	myCluster.Stop(time.Second * 5)
}

func TestCluster_Subscribe_Edges(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	myCluster := cluster.NewCluster(2, ctx)
	myCluster.Run()

	_, err := myCluster.Subscribe("non-existent-id")
	if err == nil {
		t.Error("expected error for non-existent task, but got nil")
	} else {
		t.Logf("Got expected error: %v", err)
	}

	job := func(id string, ctx context.Context, cancelled <-chan struct{}, report task.Reporter) {
		report(id, "cached-data", nil)
	}

	taskID, _ := myCluster.AddTask(task.NewTask("cache-test", job)).
		IsCacheable(true).
		Submit()

	resChan, _ := myCluster.Subscribe(taskID)
	<-resChan

	cachedChan, err := myCluster.Subscribe(taskID)
	if err != nil {
		t.Fatalf("failed to subscribe to cached task: %v", err)
	}

	select {
	case res := <-cachedChan:
		if res.Result != "cached-data" {
			t.Errorf("expected 'cached-data' from cache, got %v", res.Result)
		}
	case <-time.After(1 * time.Second):
		t.Error("timeout waiting for cached result")
	}
}

func TestCluster_TaskExecution_EdgeCases(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	myCluster := cluster.NewCluster(2, ctx)
	myCluster.Run()

	t.Run("TaskPanic", func(t *testing.T) {
		panicJob := func(id string, ctx context.Context, cancel <-chan struct{}, report task.Reporter) {
			panic("something went wrong!")
		}

		id, _ := myCluster.AddTask(task.NewTask("panic-task", panicJob)).Submit()
		resChan, _ := myCluster.Subscribe(id)

		res := <-resChan
		if res.Err == nil || !contains(res.Err.Error(), "task panicked") {
			t.Errorf("expected panic error, got: %v", res.Err)
		}
	})

	t.Run("TaskReportError", func(t *testing.T) {
		errorJob := func(id string, ctx context.Context, cancel <-chan struct{}, report task.Reporter) {
			report(id, nil, fmt.Errorf("business logic error"))
		}

		id, _ := myCluster.AddTask(task.NewTask("error-task", errorJob)).Submit()
		resChan, _ := myCluster.Subscribe(id)

		res := <-resChan
		if res.Err == nil || res.Err.Error() != "business logic error" {
			t.Errorf("expected business error, got: %v", res.Err)
		}
	})

	t.Run("TaskTimeoutDuringReport", func(t *testing.T) {
		timeoutJob := func(id string, ctx context.Context, cancel <-chan struct{}, report task.Reporter) {
			time.Sleep(200 * time.Millisecond)
			report(id, "late data", nil)
		}

		id, _ := myCluster.AddTask(task.NewTask("timeout-task", timeoutJob)).
			WithTimeout(100 * time.Millisecond).
			Submit()

		resChan, _ := myCluster.Subscribe(id)
		res := <-resChan

		if !errors.Is(res.Err, context.DeadlineExceeded) {
			t.Errorf("expected deadline exceeded, got: %v", res.Err)
		}
	})
}

func TestCluster_TaskCluster_EdgeCases(t *testing.T) {
	t.Run("ZeroWorkers", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("The code did not panic with 0 workers")
			} else {
				t.Logf("Recovered from expected panic: %v", r)
			}
		}()

		cluster.NewCluster(0, context.Background())
	})

	t.Run("NilTask", func(t *testing.T) {
		myCluster := cluster.NewCluster(2, context.Background())

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("The code did not panic when adding nil task")
			} else {
				t.Logf("Recovered from expected panic: %v", r)
			}
		}()

		myCluster.AddTask(nil)
	})
}

func TestCluster_TaskCluster_Retry(t *testing.T) {
	myCluster := cluster.NewCluster(2, context.Background())
	myCluster.Run()

	invalidError := fmt.Errorf("invalid error")
	job1 := func(id string, ctx context.Context, cancelled <-chan struct{}, report task.Reporter) {
		report(id, "Data from task-1", invalidError)
	}

	task1 := task.NewTask("", job1)
	_, _ = myCluster.AddTask(task1).
		WithRetry(5).
		WithJitter().
		RetryIf(func(err error) bool {
			return err.Error() == invalidError.Error()
		}).
		Submit()

	_, _ = myCluster.AddTask(task1).
		WithRetry(1).
		WithRetryMode(cluster.Immediate).
		WithBackoffStrategy(cluster.NewFixedBackoff(1 * time.Second)).
		Submit()

	_, _ = myCluster.AddTask(task1).
		WithRetry(1).
		WithRetryMode(cluster.Requeue).
		WithBackoffStrategy(cluster.NewLinearBackoff(1*time.Second, 1*time.Second)).
		Submit()

	_, _ = myCluster.AddTask(task1).
		WithRetry(3).
		WithRetryMode(cluster.Requeue).
		WithBackoffStrategy(cluster.NewExponentialBackoff(100*time.Millisecond, 2.0, 500*time.Millisecond)).
		Submit()

	_ = cluster.NewExponentialBackoff(1*time.Second, 0.5, 500*time.Millisecond)

	time.Sleep(5 * time.Second)
}

func TestCluster_Scheduler_Coverage_CtxDoneDuringWait(t *testing.T) {
	ctx := context.Background()

	myCluster := cluster.NewCluster(1, ctx)
	myCluster.Run()

	futureTime := time.Now().Add(1 * time.Hour)

	job := func(id string, ctx context.Context, cancelled <-chan struct{}, report task.Reporter) {
		report(id, "done", nil)
	}

	_, err := myCluster.AddTask(task.NewTask("future-task", job)).
		WithStartTime(futureTime).
		Submit()

	if err != nil {
		t.Fatalf("Failed to submit task: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	myCluster.Cancel()

	time.Sleep(50 * time.Millisecond)

	t.Log("Scheduler successfully exited from wait select via ctx.Done")
}

func TestCluster_TaskCluster_TestImmediateRetry_CancelTask(t *testing.T) {
	myCluster := cluster.NewCluster(1, context.Background())
	myCluster.Run()

	job := func(id string, ctx context.Context, cancel <-chan struct{}, report task.Reporter) {
		report(id, nil, fmt.Errorf("retry me"))
	}

	id, _ := myCluster.AddTask(task.NewTask("task-1", job)).
		WithRetry(1).
		WithRetryMode(cluster.Immediate).
		WithBackoffStrategy(cluster.FixedBackoff{Delay: 10 * time.Second}).
		Submit()

	time.Sleep(500 * time.Millisecond)

	myCluster.CancelTask(id)
	time.Sleep(500 * time.Millisecond)
}

func TestCluster_TaskCluster_TestImmediateRetry_CancelCluster(t *testing.T) {
	myCluster := cluster.NewCluster(1, context.Background())
	myCluster.Run()

	job := func(id string, ctx context.Context, cancel <-chan struct{}, report task.Reporter) {
		report(id, nil, fmt.Errorf("retry me"))
	}

	_, _ = myCluster.AddTask(task.NewTask("task-1", job)).
		WithRetry(1).
		WithRetryMode(cluster.Immediate).
		WithBackoffStrategy(cluster.FixedBackoff{Delay: 10 * time.Second}).
		OnFailure(func(id string, err error) {}).
		Submit()

	time.Sleep(500 * time.Millisecond)

	myCluster.Cancel()
	time.Sleep(500 * time.Millisecond)
}

func TestCluster_TaskCluster_TestRetry_fnExited(t *testing.T) {
	myCluster := cluster.NewCluster(1, context.Background())
	myCluster.Run()

	job := func(id string, ctx context.Context, cancel <-chan struct{}, report task.Reporter) {
	}

	_, _ = myCluster.AddTask(task.NewTask("task-1", job)).
		WithRetry(1).
		WithRetryMode(cluster.Immediate).
		WithBackoffStrategy(cluster.FixedBackoff{Delay: 10 * time.Second}).
		Submit()

	time.Sleep(500 * time.Millisecond)
}

func TestCluster_TaskCluster_TestRetry_CancelTask(t *testing.T) {
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

func TestCluster_TaskCluster_TestImmediateRetry_TimeoutDuringWait(t *testing.T) {
	myCluster := cluster.NewCluster(1, context.Background())
	myCluster.Run()

	job := func(id string, ctx context.Context, cancel <-chan struct{}, report task.Reporter) {
		report(id, nil, fmt.Errorf("trigger retry"))
	}

	_, _ = myCluster.AddTask(task.NewTask("timeout-task", job)).
		WithTimeout(1 * time.Second).
		WithRetry(1).
		WithRetryMode(cluster.Immediate).
		WithBackoffStrategy(cluster.FixedBackoff{Delay: 5 * time.Second}).
		Submit()

	time.Sleep(1500 * time.Millisecond)
}
