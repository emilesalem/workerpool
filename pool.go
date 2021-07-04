package workerpool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/emilesalem/workerpool/internal/work"
	"github.com/emilesalem/workerpool/internal/worker"
	"github.com/google/uuid"
)

const (
	minWorkers int = 5
	maxWorkers int = 5000
)

type Workerpool struct {
	workers         []worker.Worker
	workforce       chan worker.Worker
	ctx             context.Context
	load            chan int8
	dispatch        chan work.Job
	doneJobs        chan work.Job
	workerWaitGroup sync.WaitGroup
	jobRequests     sync.Map
}

func CreateWorkerpool(ctx context.Context) *Workerpool {
	return &Workerpool{
		ctx:       ctx,
		workforce: make(chan worker.Worker),
		load:      make(chan int8),
		dispatch:  make(chan work.Job, 1000000),
		doneJobs:  make(chan work.Job),
	}
}

func (p *Workerpool) Work() {
	var w sync.WaitGroup
	w.Add(2)
	go p.dispatchWork()
	go p.monitorLoad(&w)
	go p.monitorJobs(&w)
	<-p.ctx.Done()
	p.workerWaitGroup.Wait()
	close(p.doneJobs)
	w.Wait()
}

func (p *Workerpool) dispatchWork() {
	for w := range p.workforce {
		go w.Do(p.dispatch, p.doneJobs, &p.workerWaitGroup)
	}
}

func (p *Workerpool) Do(w func() interface{}) <-chan interface{} {
	c := make(chan interface{})
	go func() {
		job := work.Job{
			ID:   uuid.New(),
			Work: w,
		}
		p.load <- 1
		p.dispatch <- job
		p.jobRequests.Store(job.ID.String(), c)
	}()
	return c
}

func (p *Workerpool) monitorJobs(w *sync.WaitGroup) {
	var jobsDone int64
	var avgWorkTime int64
	for j := range p.doneJobs {
		go p.respond(j)
		jobsDone++
		avgWorkTime = int64((float64(avgWorkTime*(jobsDone-1) + int64(j.Elapsed))) / float64(jobsDone))
		p.load <- -1
	}
	close(p.load)
	fmt.Printf("requests served: %v\n", jobsDone)
	fmt.Printf("average time to work: %s\n", time.Duration(avgWorkTime))
	w.Done()
}

func (p *Workerpool) respond(job work.Job) {
	v, _ := p.jobRequests.LoadAndDelete(job.ID.String())
	c := v.(chan interface{})
	c <- job.Result
	close(c)
}

func (p *Workerpool) monitorLoad(w *sync.WaitGroup) {
	maxWorkers := 0
	totalLoad := 0
	for x := range p.load {
		totalLoad += int(x)
		if totalLoad < 0 {
			totalLoad = 0
		}
		if float32(totalLoad) > float32(len(p.workers))*1.2 {
			p.addWorker()
		} else if float32(totalLoad) < float32(len(p.workers))*0.8 {
			p.removeWorker()
		}
		if totalLoad > 500000 {
			fmt.Printf("warning workload too high (%v pending jobs)", totalLoad)
		}
		if maxWorkers < len(p.workers) {
			maxWorkers = len(p.workers)
		}
	}
	fmt.Printf("max number of concurrent workers: %v\n", maxWorkers)
	w.Done()
}

func (p *Workerpool) addWorker() {
	if len(p.workers) == maxWorkers {
		return
	}
	w := worker.CreateWorker(p.ctx)
	p.workers = append(p.workers, w)
	p.workforce <- w
}

func (p *Workerpool) removeWorker() {
	if len(p.workers) == 1 {
		return
	}
	w := p.workers[0]
	p.workers = p.workers[1:]
	w.Cancel()
}
