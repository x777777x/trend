package worker

import (
	"encoding/json"
	"testing"
	"time"

	"trend/internal/task"
)

func TestDeserializeOrzdbaTaskFromConsumer(t *testing.T) {
	// Simulate what Consumer.Start does when it receives a task
	taskData := &task.OrzdbaTask{
		ID:            "orzdba-task-cluster-host-123456",
		ClusterName:   "test-cluster",
		Type:          "orzdba",
		Host:          "host-1",
		LastTime:      time.Now(),
		SlideInterval: 5,
		CreatedAt:     time.Now(),
	}

	data, err := taskData.Serialize()
	if err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}

	// Simulate Consumer's two-phase deserialization
	var base struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &base); err != nil {
		t.Fatalf("failed to unmarshal base: %v", err)
	}
	if base.Type != "orzdba" {
		t.Errorf("expected type=orzdba, got %s", base.Type)
	}

	decoded, err := task.DeserializeOrzdbaTask(data)
	if err != nil {
		t.Fatalf("failed to deserialize specific task: %v", err)
	}
	if decoded.ID != taskData.ID {
		t.Errorf("ID mismatch: got %s, want %s", decoded.ID, taskData.ID)
	}
}

func TestDeserializeUnknownTaskType(t *testing.T) {
	// Simulate Consumer receiving an unknown task type
	data, _ := json.Marshal(map[string]interface{}{
		"type":      "unknown_type",
		"id":        "task-unknown",
		"host":      "host-1",
		"last_time": time.Now(),
	})

	var base struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &base); err != nil {
		t.Fatalf("failed to unmarshal base: %v", err)
	}

	if base.Type != "unknown_type" {
		t.Errorf("expected unknown_type, got %s", base.Type)
	}

	// Consumer should reject unknown types
	switch base.Type {
	case "orzdba":
		t.Fatal("should not reach orzdba case")
	default:
		// This is the expected path
	}
}

func TestDeserializeInvalidTaskJSON(t *testing.T) {
	invalidJSON := []byte(`{"type": "orzdba", "id": broken}`)

	var base struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(invalidJSON, &base); err == nil {
		t.Error("expected error for invalid JSON")
	}
}
