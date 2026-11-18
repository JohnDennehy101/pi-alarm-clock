package app

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/display"
)

func boardRenderWorkers(minutes int) int {
	workers := runtime.NumCPU()

	if workers > 4 {
		workers = 4
	}

	if workers > minutes {
		workers = minutes
	}

	return workers
}

func renderBatch(
	ctx context.Context,
	renderer *display.BoardRenderer,
	start time.Time,
	minutes int,
) ([][]byte, error) {
	images := make([][]byte, minutes)
	errs := make([]error, minutes)

	jobs := createBoardRenderJobs(minutes)

	renderBoardWorkers(ctx, renderer, start, jobs, images, errs)

	if err := firstRenderError(errs); err != nil {
		return nil, err
	}

	return images, nil
}

func createBoardRenderJobs(minutes int) chan int {
	jobs := make(chan int, minutes)

	for minute := 0; minute < minutes; minute++ {
		jobs <- minute
	}

	close(jobs)

	return jobs
}

func renderBoardWorkers(
	ctx context.Context,
	renderer *display.BoardRenderer,
	start time.Time,
	jobs <-chan int,
	images [][]byte,
	errs []error,
) {
	workers := boardRenderWorkers(len(images))

	var wg sync.WaitGroup

	for worker := 0; worker < workers; worker++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for minute := range jobs {
				renderMinute(ctx, renderer, start, minute, images, errs)
			}
		}()
	}

	wg.Wait()
}

func renderMinute(
	ctx context.Context,
	renderer *display.BoardRenderer,
	start time.Time,
	minute int,
	images [][]byte,
	errs []error,
) {
	renderTime := start.Add(time.Duration(minute) * time.Minute)

	images[minute], errs[minute] = renderer.RenderBoard(ctx, renderTime)
}

func firstRenderError(errs []error) error {
	for minute, err := range errs {
		if err != nil {
			return fmt.Errorf("rendering board minute %d: %w", minute, err)
		}
	}

	return nil
}
