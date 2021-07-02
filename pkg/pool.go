package workerpool

import (
	"context"
	"fmt"
)

const (
	minWorkers int = 5
	maxWorkers int = 500
)

type Worker struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func (w Worker) do(jobs chan func(), load chan int8) {
	for {
		select {
		case j := <-jobs:
			j()
			load <- -1
		case <-w.ctx.Done():
			return
		}
	}
}

type Workerpool struct {
	workers   []Worker
	workforce chan Worker
	ctx       context.Context
	load      chan int8
	jobOrders chan func()
	dispatch  chan func()
}

func CreateWorkerpool(ctx context.Context, jobOrders chan func()) *Workerpool {
	return &Workerpool{
		ctx:       ctx,
		jobOrders: jobOrders,
		dispatch:  make(chan func(), 1000000),
	}
}

func (p *Workerpool) Work() {
	go func() {
		for w := range p.workforce {
			go w.do(p.dispatch, p.load)
		}
	}()
	go p.dispatchJobs()
	go p.monitorLoad()
}

func (p *Workerpool) dispatchJobs() {
	for j := range p.jobOrders {
		p.dispatch <- j
		p.load <- 1
	}
}

func (p *Workerpool) monitorLoad() {
	totalLoad := 0
	for x := range p.load {
		totalLoad += int(x)
		if totalLoad < 0 {
			totalLoad = 0
		}
		if float32(totalLoad) > float32(len(p.workers))*1.2 {
			p.addWorker()
			fmt.Printf("load too high, adding a worker (%v pending jobs for %v workers)", totalLoad, len(p.workers))
		} else if float32(totalLoad) < float32(len(p.workers))*0.8 {
			p.removeWorker()
			fmt.Printf("load too low, removing a worker (%v pending jobs for %v workers)", totalLoad, len(p.workers))
		}
		if totalLoad > 500000 {
			fmt.Printf("warning workload too high (%v pending jobs)", totalLoad)
		}
	}
}

func (p *Workerpool) addWorker() {
	if len(p.workers) == maxWorkers {
		return
	}
	ctx, cancel := context.WithCancel(p.ctx)
	w := Worker{
		ctx,
		cancel,
	}
	p.workers = append(p.workers, w)
	p.workforce <- w
}

func (p *Workerpool) removeWorker() {
	if len(p.workers) == 1 {
		return
	}
	w := p.workers[0]
	p.workers = p.workers[1:]
	w.cancel()
}
