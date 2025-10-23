package worker

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	workerPoolDefaultSize              = 100
	taskQueueDefaultCapacityMultiplier = 100
)

type WorkerPool struct {
	tasks               chan task
	size                uint32
	taskQueueMultiplier uint32
	workers             chan struct{}
	wg                  sync.WaitGroup
}

type task struct {
	ID uuid.UUID
	fn func() error
}

type optFn func(*WorkerPool)

func WithPoolSize(size uint32) optFn {
	return func(p *WorkerPool) {
		p.size = size
		p.workers = make(chan struct{}, size)
		WithTaskQueueCapacityMultiplier(p.taskQueueMultiplier)(p)
	}
}

func WithTaskQueueCapacityMultiplier(factor uint32) optFn {
	return func(p *WorkerPool) {
		p.tasks = make(chan task, p.size*factor)
	}
}

func New(opts ...optFn) *WorkerPool {
	p := &WorkerPool{
		size:                workerPoolDefaultSize,
		tasks:               make(chan task, workerPoolDefaultSize*taskQueueDefaultCapacityMultiplier),
		workers:             make(chan struct{}, workerPoolDefaultSize),
		wg:                  sync.WaitGroup{},
		taskQueueMultiplier: taskQueueDefaultCapacityMultiplier,
	}

	for _, opt := range opts {
		opt(p)
	}

	collector := collector(p)

	p.wg.Go(func() {
		wg := sync.WaitGroup{}
		for task := range p.tasks {
			workerID := uuid.Must(uuid.NewV7())

			p.workers <- struct{}{} // acquire slot

			wg.Go(func() {
				start := time.Now()
				defer func() {
					if r := recover(); r != nil {
						slog.Error("worker panicked", "worker-id", workerID, "error", r)
					}

					<-p.workers // release slot

					took := time.Since(start)
					collector <- took

					slog.Debug("worker ended", "worker-id", workerID, "took", took.String())
				}()
				slog.Debug("worker started", "worker-id", workerID, "task-id", task.ID)
				task.fn()
			})
		}

		wg.Wait()
	})

	return p
}

func (p *WorkerPool) Dispatch(fn func() error) (taskId uuid.UUID, err error) {
	defer func() {
		r := recover()
		if r != nil {
			// named return value
			err = fmt.Errorf("tried to write to a closed channel: %s", r)
			return
		}
	}()

	taskId = uuid.Must(uuid.NewV7())
	p.tasks <- task{
		fn: fn,
		ID: taskId,
	}
	return taskId, nil
}

func (p *WorkerPool) Wait() {
	slog.Debug("closing incoming tasks channel")
	close(p.tasks)
	now := time.Now()
	slog.Debug("waiting for queued tasks", "queued-tasks", len(p.tasks))
	p.wg.Wait()
	slog.Debug("worker pool finished all tasks", "took", time.Since(now).String())
}
