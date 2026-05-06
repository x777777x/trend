package storage

import (
	"testing"
	"time"

	"trend/internal/models"
)

func TestIsValidPercentile(t *testing.T) {
	valid := []int{30, 50, 70, 90, 95, 99}
	for _, p := range valid {
		if !IsValidPercentile(p) {
			t.Errorf("IsValidPercentile(%d) = false, want true", p)
		}
	}

	invalid := []int{0, 1, 25, 33, 55, 80, 100, -1}
	for _, p := range invalid {
		if IsValidPercentile(p) {
			t.Errorf("IsValidPercentile(%d) = true, want false", p)
		}
	}
}

func TestPercentileToIndex(t *testing.T) {
	tests := []struct {
		p    int
		want int
	}{
		{99, models.IdxP99},
		{95, models.IdxP95},
		{90, models.IdxP90},
		{70, models.IdxP70},
		{50, models.IdxP50},
		{30, models.IdxP30},
		{0, -1},
		{55, -1},
		{100, -1},
	}
	for _, tt := range tests {
		got := PercentileToIndex(tt.p)
		if got != tt.want {
			t.Errorf("PercentileToIndex(%d) = %d, want %d", tt.p, got, tt.want)
		}
	}
}

func TestTrendPointMarshalJSON(t *testing.T) {
	p := TrendPoint{Timestamp: 1700000000, Value: 95.12345}
	b, err := p.MarshalJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := string(b)
	want := "[1700000000000,95.1235]"
	if got != want {
		t.Errorf("MarshalJSON = %s, want %s", got, want)
	}

	// Test zero value
	p2 := TrendPoint{Timestamp: 0, Value: 0}
	b2, _ := p2.MarshalJSON()
	if string(b2) != "[0,0.0000]" {
		t.Errorf("zero MarshalJSON = %s, want [0,0.0000]", string(b2))
	}
}

func TestFindMetricIndex(t *testing.T) {
	attrs := []models.MetricAttribute{
		{Name: "cpu_usage", Type: "float"},
		{Name: "dml", Type: "int"},
		{Name: "qps", Type: "float"},
	}

	tests := []struct {
		name string
		want int
	}{
		{"cpu_usage", 0},
		{"dml", 1},
		{"qps", 2},
		{"nonexistent", -1},
		{"", -1},
	}
	for _, tt := range tests {
		got := findMetricIndex(attrs, tt.name)
		if got != tt.want {
			t.Errorf("findMetricIndex(%q) = %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		input interface{}
		want  float64
	}{
		{float64(42.5), 42.5},
		{float64(0), 0},
		{[]byte("3.14"), 3.14},
		{[]byte("0"), 0},
		{"string", 0}, // unsupported type returns 0
		{nil, 0},
	}
	for _, tt := range tests {
		got := toFloat64(tt.input)
		if got != tt.want {
			t.Errorf("toFloat64(%v) = %f, want %f", tt.input, got, tt.want)
		}
	}
}

func TestQueryTrendDataNilDB(t *testing.T) {
	_, err := QueryTrendData("cluster-a", "orzdba", "cpu_usage", 1, 50,
		time.Now().Add(-1*time.Hour), time.Now())
	if err == nil {
		t.Error("expected error when ResultsDB is nil, got nil")
	}
}

func TestQueryTrendDataInvalidPercentile(t *testing.T) {
	_, err := QueryTrendData("cluster-a", "orzdba", "cpu_usage", 1, 55,
		time.Now().Add(-1*time.Hour), time.Now())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestQueryTrendDataMultiPercentileNilDB(t *testing.T) {
	_, err := QueryTrendDataMultiPercentile("cluster-a", "orzdba", "cpu_usage", 1, []int{50, 95},
		time.Now().Add(-1*time.Hour), time.Now())
	if err == nil {
		t.Error("expected error when ResultsDB is nil, got nil")
	}
}

func TestQueryTrendDataAllMetricsNilDB(t *testing.T) {
	_, err := QueryTrendDataAllMetrics("cluster-a", "orzdba", 1, 50,
		time.Now().Add(-1*time.Hour), time.Now())
	if err == nil {
		t.Error("expected error when ResultsDB is nil, got nil")
	}
}
