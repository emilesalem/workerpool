package worker

import (
	"context"
	"sync"
	"time"

	"github.com/emilesalem/workerpool/internal/work"
)

type Worker struct {
	ctx    context.Context
	Cancel context.CancelFunc
}

func CreateWorker(baseCtx context.Context) Worker {
	ctx, cancel := context.WithCancel(baseCtx)
	return Worker{
		ctx,
		cancel,
	}
}

func (w Worker) Do(jobs chan work.Job, doneJobs chan work.Job, wg *sync.WaitGroup) {
	wg.Add(1)
	for {
		select {
		case j, ok := <-jobs:
			if !ok {
				wg.Done()
				return
			}
			start := time.Now()
			result := j.Work()
			j.Result = result
			j.Elapsed = time.Since(start)
			doneJobs <- j
		case <-w.ctx.Done():
			wg.Done()
			return
		}
	}
}
