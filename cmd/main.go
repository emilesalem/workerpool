package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	workerpool "github.com/emilesalem/workerpool"
)

func sign() interface{} {
	time.Sleep(5 * time.Millisecond)
	return 1
}

func producer(ctx context.Context) chan func() interface{} {
	jobs := make(chan func() interface{})
	go func() {
		t := time.Tick(1 * time.Microsecond)
		for {
			select {
			case <-t:
				jobs <- sign
			case <-ctx.Done():
				close(jobs)
				return
			}
		}
	}()
	return jobs
}

func main() {
	start := time.Now()
	// duration of the test
	elapsed := 2 * time.Second

	deadline := time.Now().Add(elapsed)

	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	jobs := producer(ctx)

	wPool := workerpool.CreateWorkerpool(ctx)

	results := make([]<-chan interface{}, 100000)
	go func() {
		for _, r := range results {
			<-r
		}
	}()

	var w sync.WaitGroup
	w.Add(1)
	go func() {
		wPool.Work()
		w.Done()
	}()

	for j := range jobs {
		results = append(results, wPool.Do(j))
	}

	fmt.Printf("requested %v jobs\n", len(results))
	w.Wait()
	fmt.Printf("elapsed time %s\n", time.Since(start))
}
