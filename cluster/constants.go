package cluster

// RetryMode defines how the task should be retried after a failure.
type RetryMode int

const (
	// Requeue sends the task back to the priority queue after a failure.
	// This frees up the current worker immediately, allowing other tasks to run
	// during the backoff delay.
	Requeue RetryMode = iota

	// Immediate retries the task within the same worker goroutine.
	// The worker will block and wait for the backoff delay duration before
	// executing the task again. Use this for high-priority tasks where
	// keeping the execution slot is more important than worker availability.
	Immediate
)
