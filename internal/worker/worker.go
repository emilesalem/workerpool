package worker

import (
	"context"
	"sync"
	"time"
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

func (w Worker) Do(jobs chan func(), doneJobs chan time.Duration, wg *sync.WaitGroup) {
	wg.Add(1)
	for {
		select {
		case j, ok := <-jobs:
			if !ok {
				wg.Done()
				return
			}
			start := time.Now()
			j()
			doneJobs <- time.Since(start)
		case <-w.ctx.Done():
			wg.Done()
			return
		}
	}
}
