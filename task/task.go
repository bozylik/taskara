package task

import (
	"context"
)

type Reporter func(id string, val any, err error)

type TaskInterface interface {
	ID() string
	Fn() TaskFunc
}

type TaskFunc func(
	workerCtx context.Context,
	cancelled <-chan struct{},
	report Reporter,
)

type task struct {
	id string
	fn TaskFunc
}

func NewTask(id string, fn TaskFunc) TaskInterface {
	if id == "" {
		panic("empty task id")
	}
	return &task{id: id, fn: fn}
}

func (t *task) ID() string   { return t.id }
func (t *task) Fn() TaskFunc { return t.fn }
