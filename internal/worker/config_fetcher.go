package worker

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"trend/internal/config"
	"trend/pkg/logger"
	"trend/pkg/storage"
)

// FetchClusterConfig 从 Master API 拉取集群级配置
func FetchClusterConfig(masterAPI, clusterName string) (*config.ClusterConfig, error) {
	url := fmt.Sprintf("%s/api/v1/config/%s", masterAPI, clusterName)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cluster config from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cluster config API returned %d: %s", resp.StatusCode, string(body))
	}

	var cfg config.ClusterConfig
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode cluster config: %w", err)
	}

	return &cfg, nil
}

// InitClientsFromConfig 根据集群配置初始化各类客户端
func InitClientsFromConfig(clusterCfg *config.ClusterConfig) error {
	for _, ds := range clusterCfg.DataSources {
		switch ds.SourceType {
		case "elasticsearch":
			if err := initESFromMap(ds.Config); err != nil {
				return fmt.Errorf("failed to init ES data source %q: %w", ds.Name, err)
			}
		}
	}

	for _, s := range clusterCfg.Storages {
		switch s.SourceType {
		case "mysql":
			if err := initMySQLFromMap(s.Config); err != nil {
				return fmt.Errorf("failed to init storage %q: %w", s.Name, err)
			}
		}
	}

	return nil
}

func initESFromMap(cfg map[string]interface{}) error {
	var addresses []string
	if addrs, ok := cfg["addresses"].([]interface{}); ok {
		for _, a := range addrs {
			if s, ok := a.(string); ok {
				addresses = append(addresses, s)
			}
		}
	} else if addrs, ok := cfg["addresses"].([]string); ok {
		addresses = addrs
	}
	username, _ := cfg["username"].(string)
	password, _ := cfg["password"].(string)

	esCfg := &storage.ESConfig{
		Addresses: addresses,
		Username:  username,
		Password:  password,
	}
	return storage.InitES(esCfg)
}

func initMySQLFromMap(cfg map[string]interface{}) error {
	mysqlCfg := storage.MySQLConfig{}

	if v, ok := cfg["user"].(string); ok {
		mysqlCfg.User = v
	}
	if v, ok := cfg["password"].(string); ok {
		mysqlCfg.Password = v
	}
	if v, ok := cfg["host"].(string); ok {
		mysqlCfg.Host = v
	}
	if v, ok := cfg["port"].(float64); ok {
		mysqlCfg.Port = int(v)
	}
	if v, ok := cfg["dbname"].(string); ok {
		mysqlCfg.DBName = v
	}
	if v, ok := cfg["max_idle_conns"].(float64); ok {
		mysqlCfg.MaxIdleConns = int(v)
	}
	if v, ok := cfg["max_open_conns"].(float64); ok {
		mysqlCfg.MaxOpenConns = int(v)
	}
	if v, ok := cfg["conn_max_lifetime"].(float64); ok {
		mysqlCfg.ConnMaxLifetime = int(v)
	}
	if v, ok := cfg["debug"].(bool); ok {
		mysqlCfg.Debug = v
	}

	return storage.InitWorkerMySQL(&mysqlCfg)
}

// RetryFetchConfig 带重试地拉取集群配置，用于 Worker 启动时或出错后
func RetryFetchConfig(masterAPI, clusterName string, maxRetries int, retryInterval time.Duration) (*config.ClusterConfig, error) {
	var cfg *config.ClusterConfig
	var err error

	for i := 0; i < maxRetries; i++ {
		cfg, err = FetchClusterConfig(masterAPI, clusterName)
		if err == nil {
			logger.Info("Successfully fetched cluster config from Master",
				logger.String("master_api", masterAPI),
				logger.Int("attempt", i+1))
			return cfg, nil
		}
		logger.Warn("Failed to fetch cluster config, retrying",
			logger.Err(err), logger.Int("attempt", i+1), logger.Int("max_retries", maxRetries))
		time.Sleep(retryInterval)
	}

	return nil, fmt.Errorf("failed to fetch cluster config after %d attempts: %w", maxRetries, err)
}
