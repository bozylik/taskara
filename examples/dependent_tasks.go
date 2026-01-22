package main

import (
	"context"
	"fmt"
	"github.com/bozylik/taskara/cluster"
	"github.com/bozylik/taskara/task"
	"time"
)

/*func main() {
	// 1. Контекст кластера — корень всего дерева
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Инициализация
	myCluster := cluster.NewCluster(2, ctx)
	myCluster.Run()

	// --- ОПРЕДЕЛЕНИЯ ЗАДАЧ ---

	// Task 1: Генератор данных
	// Заметь: теперь функция принимает 4 аргумента (добавился id)
	job1 := func(id string, ctx context.Context, cancelled <-chan struct{}, report task.Reporter) {
		fmt.Printf("[%s] Working...\n", id)

		select {
		case <-ctx.Done():
			return
		case <-cancelled:
			return
		case <-time.After(1 * time.Second):
			report(id, "Data-from-Task-1", nil)
		}
	}

	// Создаем задачу с пустым ID (кластер сгенерирует его сам)
	t1 := task.NewTask("", job1)

	// САМОЕ ВАЖНОЕ: получаем сгенерированный ID из Submit
	id1, _ := myCluster.AddTask(t1).IsCacheable(true).Submit()
	fmt.Printf("Main: Task-1 submitted with ID: %s\n", id1)

	// Task 2: Зависит от Task 1. Используем id1 через замыкание.
	job2 := func(id string, ctx context.Context, cancelled <-chan struct{}, report task.Reporter) {
		fmt.Printf("[%s] Waiting for parent task: %s\n", id, id1)

		// Подписываемся на ID первой задачи
		ch, err := myCluster.Subscribe(id1)
		if err != nil {
			report(id, nil, err)
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-cancelled:
			return
		case res := <-ch:
			fmt.Printf("[%s] Success! Got from %s: %v\n", id, id1, res.Result)
			report(id, "Task-2-Finished", nil)
		}
	}

	// Task 3: Будет отменена вручную
	job3 := func(id string, ctx context.Context, cancelled <-chan struct{}, report task.Reporter) {
		fmt.Printf("[%s] Starting long process...\n", id)
		for i := 1; i <= 10; i++ {
			select {
			case <-ctx.Done(): // Сработает при таймауте или смерти кластера
				return
			case <-cancelled: // Сработает при CancelTask
				fmt.Printf("[%s] Stopped by manual cancel!\n", id)
				return
			case <-time.After(500 * time.Millisecond):
				fmt.Printf("[%s] Working... %d/10\n", id, i)
			}
		}
	}

	// --- ЗАПУСК ОСТАЛЬНЫХ ЗАДАЧ ---

	t2 := task.NewTask("", job2)
	id2, _ := myCluster.AddTask(t2).
		WithStartTime(time.Now().Add(2 * time.Second)).
		Submit()

	t3 := task.NewTask("", job3)
	id3, _ := myCluster.AddTask(t3).Submit()

	// --- МОНИТОРИНГ ---

	// 1. Подписываемся на Task-3, используя его реальный ID
	resCh3, _ := myCluster.Subscribe(id3)

	// 2. Отменяем Task-3 через 1.2 секунды
	go func() {
		time.Sleep(1200 * time.Millisecond)
		fmt.Printf("Main: Cancelling %s now...\n", id3)
		myCluster.CancelTask(id3)
	}()

	fmt.Printf("Main: Waiting for %s result...\n", id3)
	result3 := <-resCh3
	fmt.Printf("Main: %s finished. Status Error: %v\n", id3, result3.Err)

	// 3. Ждем Task-2 (зависимую)
	resCh2, _ := myCluster.Subscribe(id2)
	fmt.Printf("Main: Waiting for %s result...\n", id2)
	result2 := <-resCh2
	fmt.Printf("Main: %s finished. Result: %v\n", id2, result2.Result)

	time.Sleep(500 * time.Millisecond)
}*/

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Инициализация
	myCluster := cluster.NewCluster(2, ctx)
	myCluster.Run()

	job1 := func(id string, ctx context.Context, cancelled <-chan struct{}, report task.Reporter) {
		fmt.Printf("[%s] Working...\n", id)

		select {
		case <-cancelled:
			fmt.Printf("[%s] manual Cancelled\n", id)
			return
		case <-time.After(10 * time.Second):
			fmt.Printf("[%s] waiting 5 seconds...\n", id)
			report(id, "Data-from-Task-1", nil)
		}

		fmt.Println("hello")
	}

	task1 := task.NewTask("", job1)

	id, err := myCluster.AddTask(task1).Submit()
	if err != nil {
		fmt.Printf("[%s] error: %s\n", id, err)
	}

	time.Sleep(5 * time.Second)
	myCluster.CancelTask(id)
	time.Sleep(10 * time.Second)

}
