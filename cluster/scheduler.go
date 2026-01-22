package cluster

import (
	"container/heap"
	"context"
	"github.com/bozylik/taskara/container"
	"github.com/bozylik/taskara/task"
	"sync"
	"time"
)

type scheduler struct {
	mu sync.RWMutex

	waitingQueue container.ByTime
	readyQueue   container.ByPriority

	notify      chan struct{}
	readyNotify chan struct{}
}

func newScheduler() *scheduler {
	return &scheduler{
		notify:      make(chan struct{}, 1),
		readyNotify: make(chan struct{}, 1),
	}
}

func (s *scheduler) submitInternal(ct *clusterTask) {
	ct.status.Store(int32(task.StatusWaiting))

	item := &container.Item{
		Value:    ct,
		Priority: ct.Priority,
		StartAt:  ct.startTime,
	}

	s.mu.Lock()
	heap.Push(&s.waitingQueue, item)
	s.mu.Unlock()

	select {
	case s.notify <- struct{}{}:
	default:
	}
}

func (s *scheduler) runScheduler(ctx context.Context) {
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()

	for {
		s.mu.Lock()

		if s.waitingQueue.Len() == 0 {
			s.mu.Unlock()
			select {
			case <-ctx.Done():
				return
			case <-s.notify:
				continue
			}
		}

		item := s.waitingQueue.Peek().(*container.Item)
		ctask := item.Value.(*clusterTask)
		now := time.Now()

		if now.After(ctask.startTime) || now.Equal(ctask.startTime) {
			heap.Pop(&s.waitingQueue)
			ctask.status.Store(int32(task.StatusReady))
			heap.Push(&s.readyQueue, item)
			s.mu.Unlock()

			select {
			case s.readyNotify <- struct{}{}:
			default:
			}
			continue
		}

		wait := ctask.startTime.Sub(now)
		timer.Reset(wait)
		s.mu.Unlock()

		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		case <-s.notify:
		}
	}
}

func (s *scheduler) getNext(ctx context.Context) *clusterTask {
	for {
		s.mu.Lock()
		if s.readyQueue.Len() != 0 {
			item := heap.Pop(&s.readyQueue).(*container.Item)
			ctask := item.Value.(*clusterTask)
			ctask.status.Store(int32(task.StatusRunning))

			if s.readyQueue.Len() > 0 {
				select {
				case s.readyNotify <- struct{}{}:
				default:
				}
			}

			s.mu.Unlock()
			return ctask
		}

		s.mu.Unlock()

		select {
		case <-ctx.Done():
			return nil
		case <-s.readyNotify:
		}
	}
}
