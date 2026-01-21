package cluster

import (
	"container/heap"
	"context"
	"github.com/bozylik/taskara/container"
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
		task := item.Value.(*clusterTask)
		now := time.Now()

		if now.After(task.startTime) || now.Equal(task.startTime) {
			heap.Pop(&s.waitingQueue)
			heap.Push(&s.readyQueue, item)
			s.mu.Unlock()

			select {
			case s.readyNotify <- struct{}{}:
			default:
			}
			continue
		}

		wait := task.startTime.Sub(now)
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
			task := item.Value.(*clusterTask)

			if s.readyQueue.Len() > 0 {
				select {
				case s.readyNotify <- struct{}{}:
				default:
				}
			}

			s.mu.Unlock()
			return task
		}

		s.mu.Unlock()

		select {
		case <-ctx.Done():
			return nil
		case <-s.readyNotify:
		}
	}
}
