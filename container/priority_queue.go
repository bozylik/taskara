package container

import (
	"time"
)

// Item represents an entity stored within the priority queue.
// NOTE: This is a specific implementation of a priority queue for the task cluster.
// It is intended for internal use within this project only.
// Do not use this in your own projects as it is not a general-purpose library.
type Item struct {
	Value    any
	Priority int
	StartAt  time.Time
	Index    int
}

type PriorityQueue []*Item

type ByPriority struct{ PriorityQueue }

func (pq ByPriority) Less(i, j int) bool {
	if pq.PriorityQueue[i].Priority != pq.PriorityQueue[j].Priority {
		return pq.PriorityQueue[i].Priority > pq.PriorityQueue[j].Priority
	}
	return pq.PriorityQueue[i].StartAt.Before(pq.PriorityQueue[j].StartAt)
}

type ByTime struct{ PriorityQueue }

func (pq ByTime) Less(i, j int) bool {
	return pq.PriorityQueue[i].StartAt.Before(pq.PriorityQueue[j].StartAt)
}

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x any) {
	item := x.(*Item)
	item.Index = len(*pq)
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.Index = -1
	*pq = old[0 : n-1]
	return item
}

func (pq PriorityQueue) Peek() any {
	if len(pq) == 0 {
		return nil
	}
	return pq[0]
}
