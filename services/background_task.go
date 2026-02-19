package services

import (
	"context"
	"time"
)

type Task struct {
    cancel context.CancelFunc
}

func (t *Task) Stop() {
    t.cancel()
}

func Start(ctx context.Context, intervalSeconds int, fn func(ctx context.Context)) *Task {
    taskCtx, cancel := context.WithCancel(ctx)

    go func() {
        ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
        defer ticker.Stop()

        for {
            select {
            case <-ticker.C:
                fn(taskCtx)
            case <-taskCtx.Done():
                return
            }
        }
    }()

    return &Task{cancel: cancel}
}
