package task

type TaskStatus int

const (
	StatusIdle TaskStatus = iota
	StatusWaiting
	StatusReady
	StatusRunning
	StatusCompleted
	StatusCancelled
	StatusError
)
