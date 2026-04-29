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
	ID             string    `json:"id"`
	ClusterName    string    `json:"cluster_name"`
	Type           string    `json:"type"` // 例如：orzdba
	CreatedAt      time.Time `json:"created_at"`
	Host           string    `json:"instance_name"`
	LastTime       time.Time `json:"last_time"`
	SlideInterval  uint      `json:"slide_interval"`
	CalcInstanceID uint64    `json:"calc_instance_id"`
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
func (t *OrzdbaTask) calculateQuantiles(hitsList []interface{}, windowStart, windowEnd time.Time) {
	var dmls []float64
	var cpuUsages []float64
	var memUsages []float64
	var diskReads []float64
	var diskWrites []float64
	var netIns []float64
	var netOuts []float64

	for _, hit := range hitsList {
		hitMap, ok := hit.(map[string]interface{})
		if !ok {
			continue
		}
		sourceMap, ok := hitMap["_source"].(map[string]interface{})
		if !ok {
			continue
		}

		// dml
		if val, ok := sourceMap["dml"].(float64); ok {
			dmls = append(dmls, val)
		}
		// cpu_usage
		if val, ok := sourceMap["cpu_usage"].(float64); ok {
			cpuUsages = append(cpuUsages, val)
		}
		// mem_usage
		if val, ok := sourceMap["mem_usage"].(float64); ok {
			memUsages = append(memUsages, val)
		}
		// diskRead
		if valStr, ok := sourceMap["diskRead"].(string); ok {
			if v, err := utils.ParseSizeToBytes(valStr); err == nil {
				diskReads = append(diskReads, v)
			}
		} else if val, ok := sourceMap["diskRead"].(float64); ok {
			diskReads = append(diskReads, val)
		}
		// diskWrite
		if valStr, ok := sourceMap["diskWrite"].(string); ok {
			if v, err := utils.ParseSizeToBytes(valStr); err == nil {
				diskWrites = append(diskWrites, v)
			}
		} else if val, ok := sourceMap["diskWrite"].(float64); ok {
			diskWrites = append(diskWrites, val)
		}
		// netIn
		if valStr, ok := sourceMap["netIn"].(string); ok {
			if v, err := utils.ParseSizeToBytes(valStr); err == nil {
				netIns = append(netIns, v)
			}
		} else if val, ok := sourceMap["netIn"].(float64); ok {
			netIns = append(netIns, val)
		}
		// netOut
		if valStr, ok := sourceMap["netOut"].(string); ok {
			if v, err := utils.ParseSizeToBytes(valStr); err == nil {
				netOuts = append(netOuts, v)
			}
		} else if val, ok := sourceMap["netOut"].(float64); ok {
			netOuts = append(netOuts, val)
		}
	}

	calcAndLog := func(name string, values []float64) {
		if len(values) == 0 {
			return
		}
		sort.Float64s(values)
		p99 := algo.Quantile(values, 0.99)
		p95 := algo.Quantile(values, 0.95)
		p90 := algo.Quantile(values, 0.90)
		p70 := algo.Quantile(values, 0.70)
		p50 := algo.Quantile(values, 0.50)
		p30 := algo.Quantile(values, 0.30)

		logger.Info(fmt.Sprintf("[Orzdba Quantile] %s computed", name),
			logger.String("task_id", t.ID),
			logger.String("host", t.Host),
			logger.Float64("p99", p99),
			logger.Float64("p95", p95),
			logger.Float64("p90", p90),
			logger.Float64("p70", p70),
			logger.Float64("p50", p50),
			logger.Float64("p30", p30))

		result := &models.TrendQuantileResult{
			ClusterName: t.ClusterName,
			TaskID:      t.ID,
			Host:        t.Host,
			MetricName:  name,
			P99:         p99,
			P95:         p95,
			P90:         p90,
			P70:         p70,
			P50:         p50,
			P30:         p30,
			SampleCount: len(values),
			WindowStart: windowStart,
			WindowEnd:   windowEnd,
		}
		if err := storage.SaveQuantileResult(result, t.CalcInstanceID); err != nil {
			logger.Error("Failed to save quantile result",
				logger.String("metric", name), logger.Err(err))
		}
	}

	calcAndLog("dml", dmls)
	calcAndLog("cpu_usage", cpuUsages)
	calcAndLog("mem_usage", memUsages)
	calcAndLog("diskRead", diskReads)
	calcAndLog("diskWrite", diskWrites)
	calcAndLog("netIn", netIns)
	calcAndLog("netOut", netOuts)
}

// DeserializeOrzdbaTask 提供 OrzdbaTask 的反序列化方法
func DeserializeOrzdbaTask(data []byte) (*OrzdbaTask, error) {
	var t OrzdbaTask
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}
