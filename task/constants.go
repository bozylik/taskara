package task

// TaskStatus represents the current state of a task in its lifecycle.
type TaskStatus int

const (
	// StatusIdle is the initial state before a task is submitted or processed.
	// NOTE: Not used in the current version.
	StatusIdle TaskStatus = iota

	// StatusWaiting indicates the task is scheduled for a future time and is
	// waiting in the scheduler's timer.
	StatusWaiting

	// StatusReady means the task is currently in the priority queue,
	// waiting for an available worker.
	StatusReady

	// StatusRunning indicates the task is currently being executed by a worker.
	StatusRunning

	// StatusCompleted means the task has finished successfully with no errors.
	StatusCompleted

	// StatusCancelled indicates the task was manually stopped by the user
	// or the cluster was shut down.
	StatusCancelled

	// StatusError means the task failed after all retry attempts or encountered
	// a fatal error (e.g., timeout).
	StatusError
)
