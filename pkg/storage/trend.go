package storage

import (
	"encoding/json"
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

// PercentileToIndex 将分位值映射到 metrics_data 内层数组中的索引
func PercentileToIndex(p int) int {
	switch p {
	case 99:
		return models.IdxP99
	case 95:
		return models.IdxP95
	case 90:
		return models.IdxP90
	case 70:
		return models.IdxP70
	case 50:
		return models.IdxP50
	case 30:
		return models.IdxP30
	default:
		return -1
	}
}

// QueryTrendData 查询趋势分位值数据
// 按 calcInstanceID 路由到对应分表，利用 MySQL JSON_EXTRACT 在数据库层解析指定指标的分位值
func QueryTrendData(clusterName string, taskType string, metricName string, calcInstanceID uint64, percentile int, windowStart, windowEnd time.Time) ([]TrendSeries, error) {
	if ResultsDB == nil {
		return nil, fmt.Errorf("results database not initialized")
	}

	if !IsValidPercentile(percentile) {
		return nil, fmt.Errorf("invalid percentile: %d, must be one of 30,50,70,90,95,99", percentile)
	}

	tableName := getQuantileTableName(calcInstanceID)
	pIdx := PercentileToIndex(percentile)

	// 查询该集群任务类型的版本配置，找到 metric_name 对应的索引位置
	attrs, err := getTaskVersionAttrs(clusterName, taskType)
	if err != nil {
		return nil, fmt.Errorf("failed to get task version attributes: %w", err)
	}

	metricIdx := findMetricIndex(attrs, metricName)
	if metricIdx < 0 {
		return nil, fmt.Errorf("metric %s not found in task version attributes", metricName)
	}

	// 使用 MySQL JSON_EXTRACT 直接在数据库层解析指定位置的分位值
	// JSON_EXTRACT(metrics_data, '$[metricIdx][pIdx]')
	jsonPath := fmt.Sprintf("$[%d][%d]", metricIdx, pIdx)

	type rowResult struct {
		WindowEnd   time.Time
		ClusterName string
		Host        string
		Value       *float64
	}

	var rows []rowResult
	err = ResultsDB.Table(tableName).
		Select("window_end, cluster_name, host, CAST(JSON_EXTRACT(metrics_data, ?) AS DECIMAL) as value", jsonPath).
		Where("cluster_name = ?", clusterName).
		Where("window_end >= ? AND window_end <= ?", windowStart, windowEnd).
		Order("window_end ASC").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("query trend data failed: %w", err)
	}

	series := TrendSeries{
		Metric: map[string]string{
			"calc_instance_id": fmt.Sprintf("%d", calcInstanceID),
			"metric_name":      metricName,
			"percentile":       fmt.Sprintf("%d", percentile),
		},
		Values: make([]TrendPoint, 0, len(rows)),
	}

	for _, r := range rows {
		if len(r.ClusterName) > 0 {
			series.Metric["cluster_name"] = r.ClusterName
		}
		if len(r.Host) > 0 {
			series.Metric["instance_name"] = r.Host
		}
		if r.Value != nil {
			series.Values = append(series.Values, TrendPoint{
				Timestamp: r.WindowEnd.Unix(),
				Value:     *r.Value,
			})
		}
	}

	if len(series.Values) == 0 {
		return nil, nil
	}

	return []TrendSeries{series}, nil
}

// QueryTrendDataMultiPercentile 查询多个分位值的趋势数据
// 一次性通过多个 JSON_EXTRACT 获取多个分位值
func QueryTrendDataMultiPercentile(clusterName string, taskType string, metricName string, calcInstanceID uint64, percentiles []int, windowStart, windowEnd time.Time) ([]TrendSeries, error) {
	if ResultsDB == nil {
		return nil, fmt.Errorf("results database not initialized")
	}

	tableName := getQuantileTableName(calcInstanceID)

	attrs, err := getTaskVersionAttrs(clusterName, taskType)
	if err != nil {
		return nil, fmt.Errorf("failed to get task version attributes: %w", err)
	}

	metricIdx := findMetricIndex(attrs, metricName)
	if metricIdx < 0 {
		return nil, fmt.Errorf("metric %s not found in task version attributes", metricName)
	}

	// 构建多个 JSON_EXTRACT 列
	selectCols := "window_end, cluster_name, host"
	colAliases := make([]string, 0, len(percentiles))
	for _, p := range percentiles {
		pIdx := PercentileToIndex(p)
		if pIdx < 0 {
			return nil, fmt.Errorf("invalid percentile: %d", p)
		}
		jsonPath := fmt.Sprintf("$[%d][%d]", metricIdx, pIdx)
		colAliases = append(colAliases, fmt.Sprintf("CAST(JSON_EXTRACT(metrics_data, '%s') AS DECIMAL) as p%d", jsonPath, p))
	}
	selectCols += ", " + colAliases[0]
	for _, alias := range colAliases[1:] {
		selectCols += ", " + alias
	}

	type multiRowResult struct {
		WindowEnd   time.Time
		ClusterName string
		Host        string
		Values      map[string]*float64
	}

	var rawRows []map[string]interface{}
	err = ResultsDB.Raw("SELECT "+selectCols+" FROM "+tableName+" WHERE cluster_name = ? AND window_end >= ? AND window_end <= ? ORDER BY window_end ASC", clusterName, windowStart, windowEnd).
		Find(&rawRows).Error
	if err != nil {
		return nil, fmt.Errorf("query trend data multi-percentile failed: %w", err)
	}

	// 为每个分位值构建一个 series
	resultMap := make(map[int]*TrendSeries)
	for _, p := range percentiles {
		resultMap[p] = &TrendSeries{
			Metric: map[string]string{
				"calc_instance_id": fmt.Sprintf("%d", calcInstanceID),
				"metric_name":      metricName,
				"percentile":       fmt.Sprintf("%d", p),
			},
			Values: make([]TrendPoint, 0, len(rawRows)),
		}
	}

	for _, raw := range rawRows {
		windowEndVal, _ := raw["window_end"].(time.Time)
		clusterName, _ := raw["cluster_name"].(string)
		host, _ := raw["host"].(string)

		for _, p := range percentiles {
			s := resultMap[p]
			if len(clusterName) > 0 {
				s.Metric["cluster_name"] = clusterName
			}
			if len(host) > 0 {
				s.Metric["instance_name"] = host
			}

			colName := fmt.Sprintf("p%d", p)
			if val, ok := raw[colName]; ok && val != nil {
				f := toFloat64(val)
				s.Values = append(s.Values, TrendPoint{
					Timestamp: windowEndVal.Unix(),
					Value:     f,
				})
			}
		}
	}

	result := make([]TrendSeries, 0, len(percentiles))
	for _, p := range percentiles {
		s := resultMap[p]
		if len(s.Values) > 0 {
			result = append(result, *s)
		}
	}

	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

// QueryTrendDataAllMetrics 查询所有指标的趋势数据
// 解析整行 metrics_data JSON，按版本属性拆分
func QueryTrendDataAllMetrics(clusterName string, taskType string, calcInstanceID uint64, percentile int, windowStart, windowEnd time.Time) ([]TrendSeries, error) {
	if ResultsDB == nil {
		return nil, fmt.Errorf("results database not initialized")
	}

	if !IsValidPercentile(percentile) {
		return nil, fmt.Errorf("invalid percentile: %d", percentile)
	}

	tableName := getQuantileTableName(calcInstanceID)

	attrs, err := getTaskVersionAttrs(clusterName, taskType)
	if err != nil {
		return nil, fmt.Errorf("failed to get task version attributes: %w", err)
	}

	// 查询原始行（不解析 JSON）
	type rawRow struct {
		WindowEnd   time.Time
		ClusterName string
		Host        string
		MetricsData string
	}

	var rows []rawRow
	err = ResultsDB.Table(tableName).
		Select("window_end, cluster_name, host, metrics_data").
		Where("cluster_name = ?", clusterName).
		Where("window_end >= ? AND window_end <= ?", windowStart, windowEnd).
		Order("window_end ASC").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("query trend data all metrics failed: %w", err)
	}

	pIdx := PercentileToIndex(percentile)

	// 为每个 metric 构建 series
	seriesMap := make(map[int]*TrendSeries)
	for i, attr := range attrs {
		seriesMap[i] = &TrendSeries{
			Metric: map[string]string{
				"calc_instance_id": fmt.Sprintf("%d", calcInstanceID),
				"metric_name":      attr.Name,
				"percentile":       fmt.Sprintf("%d", percentile),
			},
			Values: make([]TrendPoint, 0, len(rows)),
		}
	}

	for _, r := range rows {
		var metricsData [][]float64
		if err := json.Unmarshal([]byte(r.MetricsData), &metricsData); err != nil {
			continue
		}
		for i, s := range seriesMap {
			if len(r.ClusterName) > 0 {
				s.Metric["cluster_name"] = r.ClusterName
			}
			if len(r.Host) > 0 {
				s.Metric["instance_name"] = r.Host
			}
			if i < len(metricsData) && pIdx < len(metricsData[i]) {
				s.Values = append(s.Values, TrendPoint{
					Timestamp: r.WindowEnd.Unix(),
					Value:     metricsData[i][pIdx],
				})
			}
		}
	}

	result := make([]TrendSeries, 0, len(attrs))
	for _, s := range seriesMap {
		if len(s.Values) > 0 {
			result = append(result, *s)
		}
	}

	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

// getTaskVersionAttrs 查询指定集群+任务类型的最新版本 attributes
func getTaskVersionAttrs(clusterName string, taskType string) ([]models.MetricAttribute, error) {
	var version models.TrendClusterTaskVersion
	err := ResultsDB.
		Where("cluster_name = ? AND task_name = ?", clusterName, taskType).
		Order("version DESC").
		First(&version).Error
	if err != nil {
		return nil, err
	}
	return version.ParseAttributes()
}

// findMetricIndex 在 attributes 列表中查找 metric_name 的索引位置
func findMetricIndex(attrs []models.MetricAttribute, metricName string) int {
	for i, attr := range attrs {
		if attr.Name == metricName {
			return i
		}
	}
	return -1
}

// toFloat64 将 interface{} 转为 float64
func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case []byte:
		var f float64
		fmt.Sscanf(string(val), "%f", &f)
		return f
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
