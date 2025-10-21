package worker

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type recorder struct {
	sync.Mutex
	durations []time.Duration
}

func (r *recorder) estimatedThroughput() (time.Duration, int) {
	r.Lock()
	defer r.Unlock()
	taskCount := len(r.durations)
	if taskCount == 0 {
		return time.Duration(0), 0
	}

	var totalTime time.Duration
	for _, duration := range r.durations {
		totalTime += duration
	}

	times, taskCount := (totalTime / time.Duration(taskCount)), taskCount

	r.durations = make([]time.Duration, 0)
	return times, taskCount
}

func collector(p *WorkerPool) chan<- time.Duration {
	collector := make(chan time.Duration)
	r := recorder{
		durations: make([]time.Duration, 0),
	}

	go func() {
		for collected := range collector {
			r.Lock()
			r.durations = append(r.durations, collected)
			r.Unlock()
		}
	}()

	go func() {
		t := time.NewTicker(15 * time.Second)
		for range t.C {
			workingWorkers := len(p.workers)
			queuedTasks := len(p.tasks)

			if workingWorkers == 0 {
				continue
			}

			var workerIds []any

			p.workerIds.Range(func(key, value any) bool {
				workerIds = append(workerIds, key)
				return true
			})

			throughput, taskCount := r.estimatedThroughput()
			if throughput == time.Duration(0) {
				continue
			}

			slog.Debug("worker pool stats",
				"active-workers", workingWorkers,
				"active-workers-ids", workerIds,
				"queued-tasks", queuedTasks,
				"estimated-throughput", fmt.Sprintf("%s/task (metric of %d tasks)", throughput, taskCount),
			)
		}
	}()

	return collector
}
