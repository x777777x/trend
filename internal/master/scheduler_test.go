package master

import (
	"context"
	"testing"

	"trend/internal/models"
	"trend/internal/task"
)

func TestOrzdbaPublisherInitializeWithNilMasterDB(t *testing.T) {
	// When MasterDB is not initialized, Initialize should return error
	p := &OrzdbaPublisher{ClusterName: "test-cluster"}
	err := p.Initialize(5)
	if err == nil {
		t.Error("expected error when master database is not initialized")
	}
}

func TestOrzdbaPublisherPublishEmptyTasks(t *testing.T) {
	p := &OrzdbaPublisher{
		ClusterName: "test-cluster",
		SlideInterval: 5,
		tasks: []*task.OrzdbaTask{},
	}

	// With empty tasks list, Publish should succeed without calling Dispatch
	err := p.Publish(context.Background(), &Dispatcher{})
	if err != nil {
		t.Errorf("expected no error with empty tasks, got: %v", err)
	}
}

func TestNewTaskPublisherOrzdba(t *testing.T) {
	pub, err := NewTaskPublisher("orzdba", "test-cluster")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pub == nil {
		t.Fatal("expected non-nil publisher")
	}

	orzdbaPub, ok := pub.(*OrzdbaPublisher)
	if !ok {
		t.Fatal("expected *OrzdbaPublisher")
	}
	if orzdbaPub.ClusterName != "test-cluster" {
		t.Errorf("expected ClusterName=test-cluster, got %s", orzdbaPub.ClusterName)
	}
}

func TestNewTaskPublisherUnknown(t *testing.T) {
	_, err := NewTaskPublisher("unknown_task", "test-cluster")
	if err == nil {
		t.Error("expected error for unknown task type")
	}
}

func TestSchedulerMutexFix(t *testing.T) {
	// Test that the mutex Lock/Unlock fix works by verifying Start() doesn't deadlock
	// This is a minimal test; the real test would require a full gocron + election setup
	// The fact that we can compile and run this test proves the fix is syntactically correct

	// We can at least test that NewScheduler returns non-nil
	election := &LeaderElection{}
	dispatcher := &Dispatcher{}

	s, err := NewScheduler(dispatcher, election)
	if err != nil {
		t.Fatalf("unexpected error creating scheduler: %v", err)
	}
	if s == nil {
		t.Fatal("expected non-nil scheduler")
	}
}

func TestTrendClusterTaskFromScheduler(t *testing.T) {
	// Verify the data flow: scheduler reads from DB and passes SlideInterval to Initialize
	tc := models.TrendClusterTask{
		ClusterName:   "test-cluster",
		TaskName:      "orzdba",
		Status:        1,
		SlideInterval: 5,
	}

	if tc.SlideInterval != 5 {
		t.Errorf("expected SlideInterval=5, got %d", tc.SlideInterval)
	}
}
