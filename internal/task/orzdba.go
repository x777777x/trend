package task

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"trend/internal/models"
	"trend/pkg/algo"
	"trend/pkg/logger"
	"trend/pkg/storage"
	"trend/pkg/utils"
)

// OrzdbaTask 具体实现了 orzdba 类型的任务
type OrzdbaTask struct {
	ID             string                   `json:"id"`
	ClusterName    string                   `json:"cluster_name"`
	Type           string                   `json:"type"` // 例如：orzdba
	CreatedAt      time.Time                `json:"created_at"`
	Host           string                   `json:"instance_name"`
	LastTime       time.Time                `json:"last_time"`
	SlideInterval  uint                     `json:"slide_interval"`
	CalcInstanceID uint64                   `json:"calc_instance_id"`
	Version        uint                     `json:"version"`
	Attributes     []models.MetricAttribute `json:"attributes"`
}

func (t *OrzdbaTask) GetID() string {
	return t.ID
}

func (t *OrzdbaTask) GetClusterName() string {
	return t.ClusterName
}

func (t *OrzdbaTask) GetType() string {
	return t.Type
}

func (t *OrzdbaTask) GetHost() string {
	return t.Host
}

func (t *OrzdbaTask) GetLastTime() time.Time {
	return t.LastTime
}

func (t *OrzdbaTask) GetSlideInterval() uint {
	return t.SlideInterval
}

func (t *OrzdbaTask) GetCalcInstanceID() uint64 {
	return t.CalcInstanceID
}

func (t *OrzdbaTask) GetVersion() uint {
	return t.Version
}

func (t *OrzdbaTask) GetAttributes() []models.MetricAttribute {
	return t.Attributes
}

// Serialize 提供 OrzdbaTask 的序列化方法
func (t *OrzdbaTask) Serialize() ([]byte, error) {
	return json.Marshal(t)
}

// Run 负责检查并执行 OrzdbaTask 具体的运算和处理逻辑
func (t *OrzdbaTask) Run() error {
	endTime := t.LastTime.Add(1 * time.Minute)

	// 1. 检查数据是否有最新
	exists, err := t.checkDataExists(endTime)
	if err != nil {
		return err
	}
	if !exists {
		logger.Info("No new data found for host in current window, skipping this run",
			logger.String("task_id", t.ID),
			logger.String("host", t.Host))
		return nil
	}

	// 2. 获取实际历史数据
	windowStart := endTime.Add(-time.Duration(t.SlideInterval) * time.Minute)
	hitsList, err := t.fetchHistoryData(windowStart, endTime)
	if err != nil {
		return err
	}
	if len(hitsList) == 0 {
		logger.Warn("ES data fetch returned 0 hits despite existence check",
			logger.String("task_id", t.ID), logger.String("host", t.Host))
		return nil
	}

	// 3. 历史数据分位值计算
	t.calculateQuantiles(hitsList, windowStart, endTime)

	return nil
}

// checkDataExists 检查 LastTime 后 1 分钟是否有新数据
func (t *OrzdbaTask) checkDataExists(endTime time.Time) (bool, error) {
	es := storage.GetES()
	if es == nil {
		return false, fmt.Errorf("elasticsearch client not initialized")
	}

	tsMultiplier, err := storage.GetTimestampFormat()
	if err != nil {
		return false, err
	}

	existsQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"term": map[string]interface{}{
							"ip": t.Host,
						},
					},
					map[string]interface{}{
						"range": map[string]interface{}{
							"timestamp": map[string]interface{}{
								"gte": t.LastTime.Unix() * tsMultiplier,
								"lte": endTime.Unix() * tsMultiplier,
							},
						},
					},
				},
			},
		},
		"size": 0,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(existsQuery); err != nil {
		return false, err
	}

	res, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex("metrics-*", "trend-orzdba-*"),
		es.Search.WithBody(&buf),
		es.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return false, fmt.Errorf("failed to check data existence in ES: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return false, fmt.Errorf("ES check search error: %s", res.String())
	}

	var existsResult map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&existsResult); err != nil {
		return false, err
	}

	hitsMap, _ := existsResult["hits"].(map[string]interface{})
	var total float64
	if totalMap, ok := hitsMap["total"].(map[string]interface{}); ok {
		total, _ = totalMap["value"].(float64)
	} else {
		total, _ = hitsMap["total"].(float64) // For older ES versions fallback
	}

	return int64(total) > 0, nil
}

// fetchHistoryData 获取计算分位值所需的历史窗口数据
func (t *OrzdbaTask) fetchHistoryData(startTime, endTime time.Time) ([]interface{}, error) {
	es := storage.GetES()
	if es == nil {
		return nil, fmt.Errorf("elasticsearch client not initialized")
	}

	tsMultiplier, err := storage.GetTimestampFormat()
	if err != nil {
		return nil, err
	}

	dataQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"term": map[string]interface{}{
							"ip": t.Host,
						},
					},
					map[string]interface{}{
						"range": map[string]interface{}{
							"timestamp": map[string]interface{}{
								"gte": startTime.Unix() * tsMultiplier,
								"lte": endTime.Unix() * tsMultiplier,
							},
						},
					},
				},
			},
		},
		"size": 10000,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(dataQuery); err != nil {
		return nil, err
	}

	resData, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex("metrics-*", "trend-orzdba-*"),
		es.Search.WithBody(&buf),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch window data from ES: %w", err)
	}
	defer resData.Body.Close()

	if resData.IsError() {
		return nil, fmt.Errorf("ES window fetch search error: %s", resData.String())
	}

	var searchResult map[string]interface{}
	if err := json.NewDecoder(resData.Body).Decode(&searchResult); err != nil {
		return nil, err
	}

	hitsMapData, _ := searchResult["hits"].(map[string]interface{})
	hitsList, _ := hitsMapData["hits"].([]interface{})

	return hitsList, nil
}

// calculateQuantiles 对给定的 ES hits 数据进行各维度指标的数据提取及分位值计算
// 按 Version.Attributes 定义的属性列表动态解析
func (t *OrzdbaTask) calculateQuantiles(hitsList []interface{}, windowStart, windowEnd time.Time) {
	if len(t.Attributes) == 0 {
		logger.Warn("No attributes defined for task, skipping quantile calculation",
			logger.String("task_id", t.ID), logger.String("host", t.Host))
		return
	}

	// 为每个属性准备一个数据切片
	dataBuckets := make([][]float64, len(t.Attributes))

	// 遍历所有 ES hits，按属性定义提取数据
	for _, hit := range hitsList {
		hitMap, ok := hit.(map[string]interface{})
		if !ok {
			continue
		}
		sourceMap, ok := hitMap["_source"].(map[string]interface{})
		if !ok {
			continue
		}

		for i, attr := range t.Attributes {
			fieldName := attr.Name
			switch attr.Type {
			case "float":
				if val, ok := sourceMap[fieldName].(float64); ok {
					dataBuckets[i] = append(dataBuckets[i], val)
				}
			case "int":
				if valStr, ok := sourceMap[fieldName].(string); ok {
					if v, err := utils.ParseSizeToBytes(valStr); err == nil {
						dataBuckets[i] = append(dataBuckets[i], v)
					}
				} else if val, ok := sourceMap[fieldName].(float64); ok {
					dataBuckets[i] = append(dataBuckets[i], val)
				}
			}
		}
	}

	// 对每个有数据的属性计算分位值，组装 metrics_data
	metricsData := make([][]float64, len(t.Attributes))
	for i, values := range dataBuckets {
		if len(values) == 0 {
			// 没有数据的属性，填充零值
			metricsData[i] = make([]float64, 7)
			continue
		}
		sort.Float64s(values)
		p99 := algo.Quantile(values, 0.99)
		p95 := algo.Quantile(values, 0.95)
		p90 := algo.Quantile(values, 0.90)
		p70 := algo.Quantile(values, 0.70)
		p50 := algo.Quantile(values, 0.50)
		p30 := algo.Quantile(values, 0.30)

		logger.Info("Quantile computed",
			logger.String("task_id", t.ID),
			logger.String("host", t.Host),
			logger.String("metric", t.Attributes[i].Name),
			logger.Float64("p99", p99),
			logger.Float64("p95", p95),
			logger.Float64("p50", p50),
			logger.Float64("p30", p30))

		metricsData[i] = []float64{p99, p95, p90, p70, p50, p30, float64(len(values))}
	}

	result := &models.TrendQuantileResult{
		ClusterName: t.ClusterName,
		TaskID:      t.ID,
		Host:        t.Host,
		Version:     t.Version,
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
	}

	// 序列化 metrics_data 为 JSON
	dataBytes, err := json.Marshal(metricsData)
	if err != nil {
		logger.Error("Failed to marshal metrics_data", logger.String("task_id", t.ID), logger.Err(err))
		return
	}
	result.MetricsData = string(dataBytes)

	if err := storage.SaveQuantileResult(result, t.CalcInstanceID); err != nil {
		logger.Error("Failed to save quantile result", logger.String("task_id", t.ID), logger.Err(err))
	}
}

// DeserializeOrzdbaTask 提供 OrzdbaTask 的反序列化方法
func DeserializeOrzdbaTask(data []byte) (*OrzdbaTask, error) {
	var t OrzdbaTask
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}
