package worker

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"

	"trend/internal/config"
	"trend/internal/task"
	"trend/pkg/logger"
)

// Executor 处理任务的具体逻辑，带有并发控制
type Executor struct {
	sem       *semaphore.Weighted
	fetcher   *DataFetcher
	waitGroup sync.WaitGroup
}

// NewExecutor 初始化任务执行器
func NewExecutor(fetcher *DataFetcher) *Executor {
	// 允许的最大并发任务数
	maxConcurrency := int64(config.Conf.Worker.Concurrency)
	if maxConcurrency <= 0 {
		maxConcurrency = 10 // 默认值
	}

	return &Executor{
		sem:     semaphore.NewWeighted(maxConcurrency),
		fetcher: fetcher,
	}
}

// Submit 提交并执行任务，阻塞如果达到最大并发数
func (e *Executor) Submit(ctx context.Context, t task.Task) {
	err := e.sem.Acquire(ctx, 1)
	if err != nil {
		logger.Error("Failed to acquire semaphore", logger.Err(err))
		return
	}

	e.waitGroup.Add(1)

	// 开一个 goroutine 进行异步工作
	go func() {
		defer e.sem.Release(1)
		defer e.waitGroup.Done()

		e.executeTask(ctx, t)
	}()
}

func (e *Executor) executeTask(ctx context.Context, t task.Task) {
	logger.Info("Executing task", logger.String("task_id", t.GetID()), logger.String("cluster", t.GetClusterName()))
	startTime := time.Now()

	err := t.Run()
	if err != nil {
		logger.Error("Failed to run task", logger.Err(err))
	}

	duration := time.Since(startTime)
	logger.Info("Task execution finished", logger.String("task_id", t.GetID()), logger.Duration("duration", duration))
}

// WaitForCompletion 可以用来等待所有的 goroutine 执行完毕（优雅关闭时）
func (e *Executor) WaitForCompletion() {
	e.waitGroup.Wait()
}
