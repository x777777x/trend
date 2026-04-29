package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"

	"trend/internal/config"
	"trend/internal/task"
	"trend/pkg/logger"
)

// Executor 处理任务的具体逻辑，带有并发控制
type Executor struct {
	sem       *semaphore.Weighted
	waitGroup sync.WaitGroup
	onFailure func(taskType, clusterName string)
}

// NewExecutor 初始化任务执行器
func NewExecutor() *Executor {
	var maxConcurrency int64
	if config.Conf.Worker.Concurrency > 0 {
		maxConcurrency = int64(config.Conf.Worker.Concurrency)
	} else {
		maxConcurrency = 10
	}

	return &Executor{
		sem: semaphore.NewWeighted(maxConcurrency),
	}
}

// NewExecutorWithConcurrency 初始化指定并发数的执行器（用于测试）
func NewExecutorWithConcurrency(maxConcurrency int64) *Executor {
	if maxConcurrency <= 0 {
		maxConcurrency = 10
	}
	return &Executor{
		sem: semaphore.NewWeighted(maxConcurrency),
	}
}

// Submit 提交并执行任务，阻塞如果达到最大并发数
func (e *Executor) Submit(ctx context.Context, t task.Task, onComplete func(err error)) {
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

		taskErr := e.executeTask(ctx, t)
		if onComplete != nil {
			onComplete(taskErr)
		}
	}()
}

func (e *Executor) executeTask(ctx context.Context, t task.Task) error {
	var taskErr error
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Task panicked", logger.String("task_id", t.GetID()), zap.Any("panic", r))
			taskErr = fmt.Errorf("task panicked: %v", r)
		}
	}()

	logger.Info("Executing task", logger.String("task_id", t.GetID()), logger.String("cluster", t.GetClusterName()))
	startTime := time.Now()

	taskErr = t.Run()
	if taskErr != nil {
		logger.Error("Failed to run task", logger.Err(taskErr))
		if e.onFailure != nil {
			e.onFailure(t.GetType(), t.GetClusterName())
		}
	}

	duration := time.Since(startTime)
	logger.Info("Task execution finished", logger.String("task_id", t.GetID()), logger.Duration("duration", duration))
	return taskErr
}

// SetOnFailure 设置任务失败回调，失败时在 executeTask 内部触发
func (e *Executor) SetOnFailure(fn func(taskType, clusterName string)) {
	e.onFailure = fn
}

// WaitForCompletion 可以用来等待所有的 goroutine 执行完毕（优雅关闭时）
func (e *Executor) WaitForCompletion() {
	e.waitGroup.Wait()
}
