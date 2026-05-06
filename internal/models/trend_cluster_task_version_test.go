package models

import (
	"testing"
)

func TestParseAttributesValidJSON(t *testing.T) {
	v := &TrendClusterTaskVersion{
		Attributes: `[{"name":"cpu_usage","type":"float"},{"name":"dml","type":"int"}]`,
	}

	attrs, err := v.ParseAttributes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(attrs) != 2 {
		t.Fatalf("expected 2 attributes, got %d", len(attrs))
	}

	if attrs[0].Name != "cpu_usage" || attrs[0].Type != "float" {
		t.Errorf("first attribute mismatch: got %+v", attrs[0])
	}
	if attrs[1].Name != "dml" || attrs[1].Type != "int" {
		t.Errorf("second attribute mismatch: got %+v", attrs[1])
	}
}

func TestParseAttributesEmptyArray(t *testing.T) {
	v := &TrendClusterTaskVersion{
		Attributes: `[]`,
	}

	attrs, err := v.ParseAttributes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(attrs) != 0 {
		t.Errorf("expected 0 attributes, got %d", len(attrs))
	}
}

func TestParseAttributesInvalidJSON(t *testing.T) {
	v := &TrendClusterTaskVersion{
		Attributes: `not json`,
	}

	_, err := v.ParseAttributes()
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestParseAttributesEmptyString(t *testing.T) {
	v := &TrendClusterTaskVersion{
		Attributes: ``,
	}

	_, err := v.ParseAttributes()
	if err == nil {
		t.Error("expected error for empty string, got nil")
	}
}

func TestTrendClusterTaskVersionTableName(t *testing.T) {
	v := TrendClusterTaskVersion{}
	if got := v.TableName(); got != "trend_cluster_task_version" {
		t.Errorf("TableName = %q, want %q", got, "trend_cluster_task_version")
	}
}

func TestMetricValueIndexesConstants(t *testing.T) {
	// Verify the expected ordering and that sample_count is last
	if IdxP99 != 0 || IdxP95 != 1 || IdxP90 != 2 || IdxP70 != 3 || IdxP50 != 4 || IdxP30 != 5 {
		t.Error("percentile index constants not in expected order")
	}
	if IdxSampleCount != 6 {
		t.Errorf("IdxSampleCount = %d, want 6", IdxSampleCount)
	}
}
