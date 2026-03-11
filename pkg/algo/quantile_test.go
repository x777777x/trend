package algo

import (
	"testing"

	"trend/pkg/models"
)

func TestCalculateEnvelope(t *testing.T) {
	// 构造测试数据
	history := []models.DataPoint{
		{Timestamp: 1, Value: 10.0},
		{Timestamp: 2, Value: 12.0},
		{Timestamp: 3, Value: 15.0},
		{Timestamp: 4, Value: 18.0},
		{Timestamp: 5, Value: 20.0},
		{Timestamp: 6, Value: 50.0}, // outlier
		{Timestamp: 7, Value: 14.0},
		{Timestamp: 8, Value: 16.0},
	}

	lower, upper := CalculateEnvelope(history)
	t.Logf("Lower Bound: %v, Upper Bound: %v", lower, upper)

	// Q1 (25% 分位数) 大约 13，Q3 大约 19
	// IQR = 6, 1.5*IQR = 9
	// lower, upper 大概是 [4, 28] 左右。50 显然是一个异常值高于 upperBound
	
	if lower > 13.0 {
		t.Errorf("Expected lower bound <= 13, got %v", lower)
	}

	if upper < 19.0 {
		t.Errorf("Expected upper bound >= 19, got %v", upper)
	}
}

func TestQuantile(t *testing.T) {
	data := []float64{1.0, 3.0, 5.0, 7.0, 9.0}
	
	q50 := Quantile(data, 0.5)
	if q50 != 5.0 {
		t.Errorf("Expected 5.0, got %v", q50)
	}

	q25 := Quantile(data, 0.25)
	if q25 != 3.0 {
		t.Errorf("Expected 3.0, got %v", q25)
	}

	q75 := Quantile(data, 0.75)
	if q75 != 7.0 {
		t.Errorf("Expected 7.0, got %v", q75)
	}
}
