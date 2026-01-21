package task

type TaskStatus int

const (
	StatusIdle TaskStatus = iota
	StatusPending
	StatusRunning
	StatusCompleted
	StatusCancelled
	StatusError
)
