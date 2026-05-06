package master

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"trend/internal/config"
	"trend/internal/models"
	"trend/pkg/logger"
	"trend/pkg/metrics"
	"trend/pkg/storage"
)

// APIServer Master 端 HTTP API 服务
type APIServer struct {
	addr string
	srv  *http.Server
}

// NewAPIServer 创建 API 服务
func NewAPIServer(addr string) *APIServer {
	if addr == "" {
		addr = ":8080"
	}
	return &APIServer{addr: addr}
}

// Start 启动 HTTP API 服务
func (a *APIServer) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/config/", a.handleGetConfig)
	mux.HandleFunc("/api/v1/trend/", a.handleGetTrend)
	mux.Handle("/metrics", metrics.Handler())

	a.srv = &http.Server{Addr: a.addr, Handler: mux}

	go func() {
		logger.Info("Starting Master API server", logger.String("addr", a.addr))
		if err := a.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Master API server failed", logger.Err(err))
		}
	}()
}

// Stop 停止 HTTP API 服务
func (a *APIServer) Stop() {
	if a.srv != nil {
		a.srv.Close()
	}
}

// handleGetConfig GET /api/v1/config/{cluster_name}
// 返回集群级基础设施配置，Worker 启动时调用
func (a *APIServer) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 提取 cluster_name 从路径 /api/v1/config/{cluster_name}
	clusterName := r.PathValue("cluster_name")
	if clusterName == "" {
		http.Error(w, "cluster_name is required", http.StatusBadRequest)
		return
	}

	cfg := config.ClusterConfig{
		Etcd:     config.Conf.Etcd,
		TaskPath: config.Conf.Master.TaskPath,
	}

	// 查询启用的数据源
	db := storage.GetDB()
	if db != nil {
		var dataSources []models.TrendDataSource
		if err := db.Where("status = ?", 1).Find(&dataSources).Error; err != nil {
			logger.Error("Failed to query data sources", logger.Err(err))
		} else {
			for _, ds := range dataSources {
				cfg.DataSources = append(cfg.DataSources, config.DataSourceConfig{
					Name:       ds.Name,
					SourceType: ds.SourceType,
					Config:     ds.Config,
				})
			}
		}

		// 查询启用的存储配置
		var storages []models.TrendStorageConfig
		if err := db.Where("status = ?", 1).Find(&storages).Error; err != nil {
			logger.Error("Failed to query storage configs", logger.Err(err))
		} else {
			for _, s := range storages {
				cfg.Storages = append(cfg.Storages, config.StorageConfig{
					Name:       s.Name,
					SourceType: s.SourceType,
					Config:     s.Config,
				})
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(cfg); err != nil {
		logger.Error("Failed to encode cluster config response", logger.Err(err))
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

// handleGetTrend GET /api/v1/trend/{task_type}
// 返回 Prometheus query_range 兼容格式的趋势数据
func (a *APIServer) handleGetTrend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	taskType := r.PathValue("task_type")
	if taskType == "" {
		http.Error(w, "task_type is required", http.StatusBadRequest)
		return
	}

	clusterName := r.URL.Query().Get("cluster_name")
	if clusterName == "" {
		http.Error(w, "cluster_name is required", http.StatusBadRequest)
		return
	}

	metricName := r.URL.Query().Get("metric_name")
	if metricName == "" {
		http.Error(w, "metric_name is required", http.StatusBadRequest)
		return
	}

	calcInstanceIDStr := r.URL.Query().Get("calc_instance_id")
	if calcInstanceIDStr == "" {
		http.Error(w, "calc_instance_id is required", http.StatusBadRequest)
		return
	}
	calcInstanceID, err := strconv.ParseUint(calcInstanceIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid calc_instance_id", http.StatusBadRequest)
		return
	}

	// 验证 calc_instance_id 属于该任务类型
	db := storage.GetDB()
	if db == nil {
		http.Error(w, "database not available", http.StatusInternalServerError)
		return
	}
	tableName := fmt.Sprintf("trend_%s_calc_instance", taskType)
	var cnt int64
	if err := db.Table(tableName).Where("id = ?", calcInstanceID).Count(&cnt).Error; err != nil {
		logger.Error("Failed to validate calc_instance_id", logger.String("task_type", taskType), logger.Err(err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if cnt == 0 {
		http.Error(w, fmt.Sprintf("calc_instance_id %d not found for task type %s", calcInstanceID, taskType), http.StatusNotFound)
		return
	}

	pStr := r.URL.Query().Get("p")
	if pStr == "" {
		http.Error(w, "p is required", http.StatusBadRequest)
		return
	}
	percentile, err := strconv.Atoi(pStr)
	if err != nil || !storage.IsValidPercentile(percentile) {
		http.Error(w, "invalid p, must be one of 30,50,70,90,95,99", http.StatusBadRequest)
		return
	}

	windowStart := time.Now().Add(-1 * time.Hour)
	windowEnd := time.Now()

	if ws := r.URL.Query().Get("window_start"); ws != "" {
		t, err := time.Parse(time.RFC3339, ws)
		if err != nil {
			http.Error(w, "invalid window_start, must be RFC3339", http.StatusBadRequest)
			return
		}
		windowStart = t
	}
	if we := r.URL.Query().Get("window_end"); we != "" {
		t, err := time.Parse(time.RFC3339, we)
		if err != nil {
			http.Error(w, "invalid window_end, must be RFC3339", http.StatusBadRequest)
			return
		}
		windowEnd = t
	}

	series, err := storage.QueryTrendData(clusterName, taskType, metricName, calcInstanceID, percentile, windowStart, windowEnd)
	if err != nil {
		logger.Error("Failed to query trend data",
			logger.String("task_type", taskType),
			logger.String("metric_name", metricName),
			logger.String("calc_instance_id", fmt.Sprintf("%d", calcInstanceID)),
			logger.Int("p", percentile),
			logger.Err(err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if series == nil {
		series = []storage.TrendSeries{}
	}

	response := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"resultType": "matrix",
			"result":     series,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("Failed to encode trend response", logger.Err(err))
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
