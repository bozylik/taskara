package container

import (
	"container/heap"
	"testing"
	"time"
)

func TestPriorityQueue_ByPriority(t *testing.T) {
	now := time.Now()
	
	items := []*Item{
		{Value: "low-priority", Priority: 1, StartAt: now.Add(time.Minute)},
		{Value: "high-priority", Priority: 10, StartAt: now.Add(time.Hour)},
		{Value: "medium-priority", Priority: 5, StartAt: now},
	}

	pq := &PriorityQueue{}
	hpq := &ByPriority{PriorityQueue: *pq}
	heap.Init(hpq)

	for _, item := range items {
		heap.Push(hpq, item)
	}

	expectedOrder := []int{10, 5, 1}

	for _, p := range expectedOrder {
		if hpq.Len() == 0 {
			t.Fatal("queue is empty unexpectedly")
		}
		item := heap.Pop(hpq).(*Item)
		if item.Priority != p {
			t.Errorf("expected priority %d, got %d", p, item.Priority)
		}
	}
}

func TestPriorityQueue_ByTime(t *testing.T) {
	now := time.Now()

	items := []*Item{
		{Value: "later", Priority: 5, StartAt: now.Add(time.Hour)},
		{Value: "sooner", Priority: 5, StartAt: now.Add(-time.Hour)},
		{Value: "now", Priority: 5, StartAt: now},
	}

	pq := &PriorityQueue{}
	hpq := &ByTime{PriorityQueue: *pq}
	heap.Init(hpq)

	for _, item := range items {
		heap.Push(hpq, item)
	}

	peeked := hpq.Peek().(*Item)
	if !peeked.StartAt.Before(now) {
		t.Error("Peek should return the earliest item")
	}

	lastTime := time.Time{}
	for hpq.Len() > 0 {
		item := heap.Pop(hpq).(*Item)
		if item.StartAt.Before(lastTime) {
			t.Errorf("items out of order: %v was popped after %v", item.StartAt, lastTime)
		}
		lastTime = item.StartAt
	}
}

func TestPriorityQueue_Empty(t *testing.T) {
	pq := PriorityQueue{}
	if pq.Peek() != nil {
		t.Error("Peek on empty queue should be nil")
	}
	if pq.Len() != 0 {
		t.Error("Length of empty queue should be 0")
	}
}

func TestPriorityQueue_SamePriority(t *testing.T) {
	now := time.Now()

	itemLater := &Item{Value: "later", Priority: 10, StartAt: now.Add(time.Hour)}
	itemSooner := &Item{Value: "sooner", Priority: 10, StartAt: now}

	pq := &PriorityQueue{}
	hpq := &ByPriority{PriorityQueue: *pq}

	heap.Push(hpq, itemLater)
	heap.Push(hpq, itemSooner)

	first := heap.Pop(hpq).(*Item)
	if first.Value != "sooner" {
		t.Errorf("Expected 'sooner' to be first due to StartAt, got %v", first.Value)
	}
}
