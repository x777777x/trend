package storage

import (
	"fmt"
	"time"

	"trend/internal/models"
)

// TrendPoint 趋势数据点：[unix_timestamp, value]
type TrendPoint struct {
	Timestamp int64   `json:"-"`
	Value     float64 `json:"-"`
}

// MarshalJSON implements Prometheus-compatible [timestamp_ms, value] format
func (p TrendPoint) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("[%d,%.4f]", p.Timestamp*1000, p.Value)), nil
}

// TrendSeries 单个趋势序列，包含标签和对应的数据点
type TrendSeries struct {
	Metric map[string]string `json:"metric"`
	Values []TrendPoint      `json:"values"`
}

// QueryTrendData 查询趋势分位值数据
// 按 calcInstanceID 路由到对应分表，返回指定分位值的时间序列
func QueryTrendData(metricName string, calcInstanceID uint64, percentile int, windowStart, windowEnd time.Time) ([]TrendSeries, error) {
	if ResultsDB == nil {
		return nil, fmt.Errorf("results database not initialized")
	}

	if !IsValidPercentile(percentile) {
		return nil, fmt.Errorf("invalid percentile: %d, must be one of 30,50,70,90,95,99", percentile)
	}

	tableName := getQuantileTableName(calcInstanceID)

	var results []models.TrendQuantileResult

	err := ResultsDB.Table(tableName).
		Where("metric_name = ? AND window_end >= ? AND window_end <= ?", metricName, windowStart, windowEnd).
		Order("window_end ASC").
		Find(&results).Error
	if err != nil {
		return nil, fmt.Errorf("query trend data failed: %w", err)
	}

	series := TrendSeries{
		Metric: map[string]string{
			"calc_instance_id": fmt.Sprintf("%d", calcInstanceID),
			"metric_name":      metricName,
		},
		Values: make([]TrendPoint, 0, len(results)),
	}

	for _, r := range results {
		if len(r.ClusterName) > 0 {
			series.Metric["cluster_name"] = r.ClusterName
		}
		if len(r.Host) > 0 {
			series.Metric["instance_name"] = r.Host
		}

		val := percentileValue(&r, percentile)
		series.Values = append(series.Values, TrendPoint{
			Timestamp: r.WindowEnd.Unix(),
			Value:     val,
		})
	}

	if len(series.Values) == 0 {
		return nil, nil
	}

	return []TrendSeries{series}, nil
}

func percentileColumn(p int) string {
	switch p {
	case 99:
		return "p99"
	case 95:
		return "p95"
	case 90:
		return "p90"
	case 70:
		return "p70"
	case 50:
		return "p50"
	case 30:
		return "p30"
	default:
		return ""
	}
}

func percentileValue(r *models.TrendQuantileResult, p int) float64 {
	switch p {
	case 99:
		return r.P99
	case 95:
		return r.P95
	case 90:
		return r.P90
	case 70:
		return r.P70
	case 50:
		return r.P50
	case 30:
		return r.P30
	default:
		return 0
	}
}

// IsValidPercentile 校验分位值是否合法
func IsValidPercentile(p int) bool {
	switch p {
	case 30, 50, 70, 90, 95, 99:
		return true
	default:
		return false
	}
}
