package task

import (
	"encoding/json"
	"testing"
	"time"

	"trend/internal/models"
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

func TestOrzdbaTaskGetVersion(t *testing.T) {
	task := &OrzdbaTask{Version: 3}
	if task.GetVersion() != 3 {
		t.Errorf("expected version 3, got %d", task.GetVersion())
	}
}

func TestOrzdbaTaskGetCalcInstanceID(t *testing.T) {
	task := &OrzdbaTask{CalcInstanceID: 42}
	if task.GetCalcInstanceID() != 42 {
		t.Errorf("expected CalcInstanceID 42, got %d", task.GetCalcInstanceID())
	}
}

func TestOrzdbaTaskGetAttributes(t *testing.T) {
	attrs := []models.MetricAttribute{
		{Name: "cpu_usage", Type: "float"},
		{Name: "dml", Type: "int"},
	}
	task := &OrzdbaTask{Attributes: attrs}
	got := task.GetAttributes()
	if len(got) != 2 {
		t.Fatalf("expected 2 attributes, got %d", len(got))
	}
	if got[0].Name != "cpu_usage" || got[0].Type != "float" {
		t.Errorf("first attribute mismatch: got %+v", got[0])
	}
	if got[1].Name != "dml" || got[1].Type != "int" {
		t.Errorf("second attribute mismatch: got %+v", got[1])
	}
}

func TestOrzdbaTaskSerializeDeserializeRoundtripWithVersion(t *testing.T) {
	attrs := []models.MetricAttribute{
		{Name: "qps", Type: "float"},
	}
	original := &OrzdbaTask{
		ID:             "rt-test-001",
		ClusterName:    "cluster-x",
		Type:           "orzdba",
		Host:           "10.0.0.5",
		LastTime:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		SlideInterval:  5,
		CalcInstanceID: 7,
		Version:        2,
		Attributes:     attrs,
	}

	data, err := original.Serialize()
	if err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}

	decoded, err := DeserializeOrzdbaTask(data)
	if err != nil {
		t.Fatalf("failed to deserialize: %v", err)
	}

	if decoded.Version != original.Version {
		t.Errorf("Version mismatch: got %d, want %d", decoded.Version, original.Version)
	}
	if decoded.CalcInstanceID != original.CalcInstanceID {
		t.Errorf("CalcInstanceID mismatch: got %d, want %d", decoded.CalcInstanceID, original.CalcInstanceID)
	}
	if len(decoded.Attributes) != len(original.Attributes) {
		t.Fatalf("Attributes length mismatch: got %d, want %d", len(decoded.Attributes), len(original.Attributes))
	}
	if decoded.Attributes[0].Name != original.Attributes[0].Name {
		t.Errorf("Attribute name mismatch: got %s, want %s", decoded.Attributes[0].Name, original.Attributes[0].Name)
	}
}

func TestOrzdbaTaskGetAttributesEmpty(t *testing.T) {
	task := &OrzdbaTask{}
	got := task.GetAttributes()
	if got != nil {
		t.Errorf("expected nil attributes, got %+v", got)
	}
}
