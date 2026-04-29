package storage

import (
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
	// When WorkerDB is not initialized, SaveQuantileResult should return an error
	result := &models.TrendQuantileResult{
		ClusterName: "test-cluster",
		TaskID:      "task-1",
		Host:        "host-1",
		MetricName:  "cpu_usage",
		P99:         95.5,
		P95:         90.0,
		P90:         85.0,
		P70:         70.0,
		P50:         50.0,
		P30:         30.0,
		SampleCount: 100,
	}

	err := SaveQuantileResult(result, 1)
	if err == nil {
		t.Error("expected error when WorkerDB is uninitialized, got nil")
	}
}

func TestQuantileResultModelFields(t *testing.T) {
	result := models.TrendQuantileResult{
		ClusterName: "cluster-a",
		TaskID:      "task-001",
		Host:        "192.168.1.10",
		MetricName:  "dml",
		P99:         1000.5,
		P95:         900.0,
		P90:         800.0,
		P70:         500.0,
		P50:         300.0,
		P30:         100.0,
		SampleCount: 5000,
	}

	if result.ClusterName != "cluster-a" {
		t.Errorf("ClusterName mismatch")
	}
	if result.MetricName != "dml" {
		t.Errorf("MetricName mismatch")
	}
	if result.P99 != 1000.5 {
		t.Errorf("P99 mismatch: got %f", result.P99)
	}
	if result.SampleCount != 5000 {
		t.Errorf("SampleCount mismatch: got %d", result.SampleCount)
	}
}
