package workerpool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/emilesalem/workerpool/internal/worker"
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
	jobOrders       chan func()
	dispatch        chan func()
	doneJobs        chan time.Duration
	workerWaitGroup sync.WaitGroup
}

func CreateWorkerpool(ctx context.Context, jobOrders chan func()) *Workerpool {
	return &Workerpool{
		ctx:       ctx,
		jobOrders: jobOrders,
		workforce: make(chan worker.Worker),
		load:      make(chan int8),
		dispatch:  make(chan func(), 1000000),
		doneJobs:  make(chan time.Duration),
	}
}

func (p *Workerpool) Work() {
	go p.workersWork()
	go p.monitorLoad()
	go p.dispatchJobs()
	go p.monitorJobs()
	<-p.ctx.Done()
	p.workerWaitGroup.Wait()
	close(p.doneJobs)
}

func (p *Workerpool) workersWork() {
	for w := range p.workforce {
		go w.Do(p.dispatch, p.doneJobs, &p.workerWaitGroup)
	}
}

func (p *Workerpool) dispatchJobs() {
	requests := 0
	for j := range p.jobOrders {
		requests++
		p.load <- 1
		p.dispatch <- j
	}
	close(p.dispatch)
	fmt.Printf("jobs requested: %v\n", requests)
}

func (p *Workerpool) monitorJobs() {
	var jobsDone int64
	var avgWorkTime int64
	for timeWorked := range p.doneJobs {
		jobsDone++
		avgWorkTime = int64((float64(avgWorkTime*(jobsDone-1) + int64(timeWorked))) / float64(jobsDone))
		p.load <- -1
	}
	close(p.load)
	fmt.Printf("requests served: %v\n", jobsDone)
	fmt.Printf("average time to work: %s\n", time.Duration(avgWorkTime))
}

func (p *Workerpool) monitorLoad() {
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
