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
	workerIds           sync.Map
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
		workerIds:           sync.Map{},
	}

	for _, opt := range opts {
		opt(p)
	}

	collector := collector(p)

	go func() {
		for task := range p.tasks {
			workerID := uuid.Must(uuid.NewV7())

			p.workers <- struct{}{} // acquire slot
			p.workerIds.Store(workerID, struct{}{})

			p.wg.Go(func() {
				start := time.Now()
				defer func() {
					if r := recover(); r != nil {
						slog.Error("worker panicked", "worker-id", workerID, "error", r)
					}

					p.workerIds.Delete(workerID)
					<-p.workers // release slot

					took := time.Since(start)
					collector <- took

					slog.Debug("worker ended", "worker-id", workerID, "took", took.String())
				}()
				slog.Debug("worker started", "worker-id", workerID, "task-id", task.ID)
				task.fn()
			})
		}
	}()

	return p
}

func (p *WorkerPool) Dispatch(fn func() error) (taskId uuid.UUID, err error) {
	defer func() {
		r := recover()
		if r != nil {
			err = fmt.Errorf("recovered panic: %s", r)
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
	slog.Info("closing worker")
	close(p.tasks)
	slog.Info("waiting for finishing queued tasks")
	now := time.Now()
	p.wg.Wait()
	slog.Info("done finishing queued tasks", "took", time.Since(now).String())
}
