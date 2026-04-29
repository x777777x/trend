package task

import (
	"encoding/json"
	"testing"
	"time"
)

func TestOrzdbaTaskSerializeDeserialize(t *testing.T) {
	original := &OrzdbaTask{
		ID:            "test-001",
		ClusterName:   "test-cluster",
		Type:          "orzdba",
		Host:          "192.168.1.10",
		LastTime:      time.Now(),
		SlideInterval: 5,
		CreatedAt:     time.Now(),
	}

	data, err := original.Serialize()
	if err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}

	decoded, err := DeserializeOrzdbaTask(data)
	if err != nil {
		t.Fatalf("failed to deserialize: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID mismatch: got %s, want %s", decoded.ID, original.ID)
	}
	if decoded.ClusterName != original.ClusterName {
		t.Errorf("ClusterName mismatch: got %s, want %s", decoded.ClusterName, original.ClusterName)
	}
	if decoded.Type != original.Type {
		t.Errorf("Type mismatch: got %s, want %s", decoded.Type, original.Type)
	}
	if decoded.Host != original.Host {
		t.Errorf("Host mismatch: got %s, want %s", decoded.Host, original.Host)
	}
	if decoded.SlideInterval != original.SlideInterval {
		t.Errorf("SlideInterval mismatch: got %d, want %d", decoded.SlideInterval, original.SlideInterval)
	}
}

func TestDeserializeInvalidJSON(t *testing.T) {
	_, err := DeserializeOrzdbaTask([]byte("not valid json"))
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestOrzdbaTaskGetType(t *testing.T) {
	task := &OrzdbaTask{Type: "orzdba"}
	if task.GetType() != "orzdba" {
		t.Errorf("expected orzdba, got %s", task.GetType())
	}
}

func TestOrzdbaTaskGetID(t *testing.T) {
	task := &OrzdbaTask{ID: "unique-id"}
	if task.GetID() != "unique-id" {
		t.Errorf("expected unique-id, got %s", task.GetID())
	}
}

func TestOrzdbaTaskGetClusterName(t *testing.T) {
	task := &OrzdbaTask{ClusterName: "my-cluster"}
	if task.GetClusterName() != "my-cluster" {
		t.Errorf("expected my-cluster, got %s", task.GetClusterName())
	}
}

func TestOrzdbaTaskGetHost(t *testing.T) {
	task := &OrzdbaTask{Host: "host-1"}
	if task.GetHost() != "host-1" {
		t.Errorf("expected host-1, got %s", task.GetHost())
	}
}

func TestOrzdbaTaskGetSlideInterval(t *testing.T) {
	task := &OrzdbaTask{SlideInterval: 10}
	if task.GetSlideInterval() != 10 {
		t.Errorf("expected 10, got %d", task.GetSlideInterval())
	}
}

func TestOrzdbaTaskGetLastTime(t *testing.T) {
	now := time.Now()
	task := &OrzdbaTask{LastTime: now}
	if !task.GetLastTime().Equal(now) {
		t.Errorf("expected %v, got %v", now, task.GetLastTime())
	}
}

func TestOrzdbaTaskSerializeIsValidJSON(t *testing.T) {
	task := &OrzdbaTask{
		ID:            "test-002",
		ClusterName:   "cluster-a",
		Type:          "orzdba",
		Host:          "10.0.0.1",
		LastTime:      time.Now(),
		SlideInterval: 15,
		CreatedAt:     time.Now(),
	}

	data, err := task.Serialize()
	if err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("serialized data is not valid JSON: %v", err)
	}

	if m["id"] != "test-002" {
		t.Errorf("expected id=test-002 in JSON, got %v", m["id"])
	}
}
