package task

import (
	"context"
	"testing"
)

func TestTask(t *testing.T) {
	testFn := func(id string, ctx context.Context, cancelled <-chan struct{}, report Reporter) {}

	t.Run("NewTask and Getters", func(t *testing.T) {
		id := "test-id"
		tsk := NewTask(id, testFn)

		if tsk.ID() != id {
			t.Errorf("expected ID %s, got %s", id, tsk.ID())
		}

		if tsk.Fn() == nil {
			t.Error("expected TaskFunc to be not nil")
		}
	})

	t.Run("SetID", func(t *testing.T) {
		tsk := NewTask("initial-id", testFn)
		newID := "updated-id"

		tsk.SetID(newID)

		if tsk.ID() != newID {
			t.Errorf("expected updated ID %s, got %s", newID, tsk.ID())
		}
	})

	t.Run("Clone", func(t *testing.T) {
		id := "test-id"
		original := NewTask(id, testFn)
		cloned := original.Clone()

		if cloned.ID() != original.ID() {
			t.Errorf("expected ID %s, got %s", original.ID(), cloned.ID())
		}

		// Проверка, что это разные объекты в памяти
		if cloned == original {
			t.Error("Clone returned the same pointer, expected a new instance")
		}
	})
}
