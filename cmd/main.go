package main

import (
	"context"
	"fmt"
	"time"

	workerpool "github.com/emilesalem/workerpool"
)

func sign() {
	time.Sleep(5 * time.Millisecond)
}

func producer(ctx context.Context) chan func() {
	jobs := make(chan func())
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

	wPool := workerpool.CreateWorkerpool(ctx, jobs)

	wPool.Work()

	fmt.Printf("elapsed time %s\n", time.Since(start))
}
