package worker

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"trend/internal/models"
)

func TestExecutorSingleTask(t *testing.T) {
	ex := NewExecutorWithConcurrency(5)

	var executed int32
	testTask := &mockTask{
		id:          "task-1",
		clusterName: "test",
		host:        "localhost",
		runFn: func() error {
			atomic.StoreInt32(&executed, 1)
			return nil
		},
	}

	ex.Submit(context.Background(), testTask, nil)
	ex.WaitForCompletion()

	if atomic.LoadInt32(&executed) != 1 {
		t.Error("expected task to be executed")
	}
}

func TestExecutorConcurrencyLimit(t *testing.T) {
	ex := NewExecutorWithConcurrency(10)

	var concurrent int32
	var maxConcurrent int32
	done := make(chan struct{})

	testTask := &mockTask{
		id:          "task-slow",
		clusterName: "test",
		host:        "localhost",
		runFn: func() error {
			c := atomic.AddInt32(&concurrent, 1)
			atomic.StoreInt32(&maxConcurrent, c)
			time.Sleep(50 * time.Millisecond)
			atomic.AddInt32(&concurrent, -1)
			return nil
		},
	}

	// Submit more tasks than the default concurrency (10)
	for i := 0; i < 20; i++ {
		ex.Submit(context.Background(), testTask, nil)
	}

	// Give goroutines time to start
	go func() {
		time.Sleep(100 * time.Millisecond)
		close(done)
	}()

	<-done
	ex.WaitForCompletion()

	if atomic.LoadInt32(&maxConcurrent) > 10 {
		t.Errorf("concurrency exceeded: max=%d, limit=10", atomic.LoadInt32(&maxConcurrent))
	}
}

func TestExecutorContextCancelled(t *testing.T) {
	ex := NewExecutorWithConcurrency(5)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	testTask := &mockTask{
		id:          "task-cancelled",
		clusterName: "test",
		host:        "localhost",
		runFn: func() error {
			t.Error("should not execute when context is cancelled")
			return nil
		},
	}

	ex.Submit(ctx, testTask, nil)
	// Should not block forever
	time.Sleep(50 * time.Millisecond)
}

func TestExecutorMultipleTasks(t *testing.T) {
	ex := NewExecutorWithConcurrency(5)

	var count int32

	for i := 0; i < 5; i++ {
		testTask := &mockTask{
			id:          "task-multi",
			clusterName: "test",
			host:        "localhost",
			runFn: func() error {
				atomic.AddInt32(&count, 1)
				return nil
			},
		}
		ex.Submit(context.Background(), testTask, nil)
	}

	ex.WaitForCompletion()

	if atomic.LoadInt32(&count) != 5 {
		t.Errorf("expected 5 executions, got %d", atomic.LoadInt32(&count))
	}
}

// mockTask implements task.Task for testing
type mockTask struct {
	id          string
	clusterName string
	host        string
	lastTime    time.Time
	slide       uint
	runFn       func() error
}

func (m *mockTask) GetID() string                { return m.id }
func (m *mockTask) GetClusterName() string        { return m.clusterName }
func (m *mockTask) GetHost() string               { return m.host }
func (m *mockTask) GetLastTime() time.Time        { return m.lastTime }
func (m *mockTask) GetType() string               { return "mock" }
func (m *mockTask) GetSlideInterval() uint        { return m.slide }
func (m *mockTask) GetCalcInstanceID() uint64             { return 0 }
func (m *mockTask) GetVersion() uint                      { return 0 }
func (m *mockTask) GetAttributes() []models.MetricAttribute { return nil }
func (m *mockTask) Serialize() ([]byte, error)            { return nil, nil }
func (m *mockTask) Run() error                            { return m.runFn() }
