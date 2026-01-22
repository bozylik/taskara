# taskara

### lightweight task scheduler with priority and worker pool

taskara is a simple and fast library for managing tasks in go. it allows you to run many tasks concurrently using a controlled number of workers. it is designed to be minimal, having zero external dependencies.

---

> [!IMPORTANT]
> ## alpha-build
> this is an early alpha version of the project. it does not include detailed instructions or many examples yet.<br>
> **status:** under active development.<br>
>**future:** we plan to add more features, better documentation, and complex examples soon.<br>
>we do not guarantee backward compatibility of various releases until the first stable version is available.

---

## features

* **worker pool:** limit the number of active goroutines to save resources.
* **priority queue:** important tasks are executed first.
* **scheduling:** start tasks immediately or at a specific time.
* **timeouts:** automatically cancel tasks that take too long.
* **individual control:** cancel a specific task by its id without stopping others.
* **easy integration:** get results through go channels.

---

## architecture

the system consists of three main parts:
1. **scheduler:** manages the waiting and ready queues using a priority heap.
2. **executor:** runs a fixed pool of workers that process tasks.
3. **cluster:** provides a simple interface to submit and manage tasks.

