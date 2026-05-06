package storage

import (
	"encoding/json"
	"testing"

	"trend/internal/models"
)

func TestGetQuantileTableName(t *testing.T) {
	table := getQuantileTableName(1)
	if len(table) != len("metric_features_00") {
		t.Errorf("unexpected table name length: %s", table)
	}
	expectedPrefix := "metric_features_"
	if table[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("table name %q does not have expected prefix %q", table, expectedPrefix)
	}
}

func TestGetQuantileTableNameDeterministic(t *testing.T) {
	name1 := getQuantileTableName(123)
	name2 := getQuantileTableName(123)
	if name1 != name2 {
		t.Errorf("getTableName not deterministic: %s != %s", name1, name2)
	}
}

func TestSaveQuantileResultNilDB(t *testing.T) {
	metricsData := [][]float64{
		{95.5, 90.0, 85.0, 70.0, 50.0, 30.0, 100},
		{1000.5, 900.0, 800.0, 500.0, 300.0, 100.0, 5000},
	}
	dataBytes, _ := json.Marshal(metricsData)

	result := &models.TrendQuantileResult{
		ClusterName: "test-cluster",
		TaskID:      "task-1",
		Host:        "host-1",
		Version:     1,
		MetricsData: string(dataBytes),
	}

	err := SaveQuantileResult(result, 1)
	if err == nil {
		t.Error("expected error when WorkerDB is uninitialized, got nil")
	}
}

func TestQuantileResultModelFields(t *testing.T) {
	metricsData := [][]float64{
		{1000.5, 900.0, 800.0, 500.0, 300.0, 100.0, 5000},
	}
	dataBytes, _ := json.Marshal(metricsData)

	result := models.TrendQuantileResult{
		ClusterName: "cluster-a",
		TaskID:      "task-001",
		Host:        "192.168.1.10",
		Version:     2,
		MetricsData: string(dataBytes),
	}

	if result.ClusterName != "cluster-a" {
		t.Errorf("ClusterName mismatch")
	}
	if result.Version != 2 {
		t.Errorf("Version mismatch: got %d", result.Version)
	}

	var parsed [][]float64
	if err := json.Unmarshal([]byte(result.MetricsData), &parsed); err != nil {
		t.Fatalf("failed to parse MetricsData: %v", err)
	}
	if len(parsed) != 1 || len(parsed[0]) != 7 {
		t.Errorf("MetricsData shape unexpected: got %d metrics, %d values", len(parsed), len(parsed[0]))
	}
	if parsed[0][0] != 1000.5 {
		t.Errorf("MetricsData p99 mismatch: got %f", parsed[0][0])
	}
	if parsed[0][6] != 5000 {
		t.Errorf("MetricsData sample_count mismatch: got %f", parsed[0][6])
	}
}
