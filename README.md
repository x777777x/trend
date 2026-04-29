# Trend

分布式指标数据采集与分析平台，用于从 Elasticsearch 等数据源定时拉取业务指标，计算分位值（p99/p95/p90 等），并提供 Prometheus 兼容的时序查询 API。

## 架构

```
┌──────────────┐          ┌──────────────────────────────────┐          ┌──────────────────┐
│  Master xN   │          │              Etcd                │          │  Worker xN       │
│              │          │                                  │          │                  │
│ ┌──────────┐ │  发布    │  ┌────────────┐  ┌────────────┐  │  订阅    │ ┌──────────────┐ │
│ │Scheduler │ │ ───────▶ │  │ /tasks/... │  │ /workers/… │  │ ◀────── │ │Consumer      │ │
│ └──────────┘ │          │  └────────────┘  └────────────┘  │          │ └──────────────┘ │
│ ┌──────────┐ │          └──────────────────────────────────┘          │ ┌──────────────┐ │
│ │LeaderElec│ │                                                        │ │Executor      │ │
│ └──────────┘ │                                                        │ └──────────────┘ │
│ ┌──────────┐ │                                                        └───────┬──────────┘
│ │API Server│ │                                                                │
│ └──────────┘ │                                     ┌────────────┐  ┌─────────▼──────────┐
└──────┬───────┘                                     │ MySQL      │  │  Elasticsearch     │
       │                                             │ (配置/结果) │  │  (原始指标数据)    │
       │ GET /api/v1/trend/{type}                    └────────────┘  └────────────────────┘
       │ GET /api/v1/config/{cluster}
       ▼
  Prometheus / Grafana
```

**数据流**：

1. Master 只有 Leader 节点会按 cron 周期从 MySQL 读取任务配置，发布到 etcd 队列
2. Worker 订阅 etcd 任务队列，从 Elasticsearch 拉取原始指标数据
3. Worker 计算分位值后写入 MySQL 分表 `metric_features_00` ~ `metric_features_09`
4. 查询端通过 Master API 的 `/api/v1/trend/` 接口读取趋势数据

## 功能

- **分布式调度**：基于 Etcd Leader 选举 + gocron 定时触发，单 Leader 下发，多 Worker 并行执行
- **配置热加载**：Worker 启动时从 Master API 获取集群级配置（数据源、存储、etcd 地址），无需本地配置文件
- **分位值计算**：从 ES 拉取原始指标（DML/CPU/内存/磁盘I/O/网络），计算 p99/p95/p90/p70/p50/p30
- **分表存储**：结果按 `calc_instance_id % 10` 路由到 `metric_features_00` ~ `metric_features_09`
- **背压控制**：Master 端按 Worker 负载动态调节任务分发速率
- **Prometheus 监控**：Master/Worker 各自暴露存活状态、超时任务数、任务失败数等指标
- **趋势查询 API**：Prometheus query_range 兼容格式的时序数据查询

## 技术栈

| 组件 | 说明 |
|------|------|
| Go 1.25 | 编程语言 |
| MySQL + GORM | 任务配置存储、分位值结果分表 |
| Elasticsearch v8 | 原始指标数据来源 |
| Etcd v3.6 | Leader 选举 + 任务队列 |
| gocron/v2 | 定时调度 |
| Prometheus client_golang | 指标暴露 |
| Zap + lumberjack | 日志（分级 + 滚动切分） |
| Viper | 配置管理 |

## 项目结构

```
trend/
├── cmd/
│   ├── master/          # Master 入口
│   └── worker/          # Worker 入口
├── configs/             # 配置文件
├── docs/
│   └── tables.md        # 数据库表结构文档
├── internal/
│   ├── config/          # Viper 配置映射
│   ├── master/          # 调度器、任务发布者、Leader 选举、API 服务、超时采集器
│   ├── models/          # GORM 数据模型
│   ├── task/            # 具体任务实现（orzdba 等）
│   └── worker/          # 消费者、执行器、配置拉取
├── pkg/
│   ├── algo/            # 统计算法（分位值）
│   ├── etcd/            # Etcd 客户端封装
│   ├── logger/          # 日志模块（Zap + lumberjack）
│   ├── metrics/         # Prometheus 指标注册
│   ├── models/          # 公共数据结构
│   ├── storage/         # ES 客户端、MySQL 连接池、分位值存储、趋势查询
│   └── utils/           # 工具函数
├── go.mod
└── bin/                 # 编译产物
```

## 快速开始

### 1. 环境准备

- Go 1.25+
- MySQL（建库后表由 GORM AutoMigrate 自动创建）
- Elasticsearch v8
- Etcd

### 2. 编译

```bash
go build -o bin/trend-master ./cmd/master
go build -o bin/trend-worker ./cmd/worker
```

### 3. 配置

在 `configs/config.yaml` 中配置各组件连接参数：

```yaml
app:
  cluster_name: prod-cluster-1

etcd:
  endpoints:
    - http://10.10.10.30:2379
  prefix: trend

logger:
  level: info
  output: stdout  # 或文件路径

master:
  mysql:
    host: 10.10.10.40
    port: 3306
    dbname: trend_config
    user: root
    password: "..."
    max_idle_conns: 10
    max_open_conns: 100
    conn_max_lifetime: 1
  api_addr: :8080
  cron_expr: "*/1 * * * *"
  task_path: /tasks
  backlog_threshold: 100

worker:
  master_api: http://127.0.0.1:8080
  concurrency: 10
  metrics_addr: :9090
```

### 4. 初始化数据

```sql
-- 任务类型
INSERT INTO trend_cluster_task (cluster_name, task_name, status, slide_interval, stale_threshold_minutes) VALUES
('prod-cluster-1', 'orzdba', 1, 5, 30);

-- 监控实例
INSERT INTO trend_orzdba_calc_instance (cluster_name, instance_name, status, last_time) VALUES
('prod-cluster-1', '10.0.1.5', 1, NOW());

-- 数据源（ES）
INSERT INTO trend_data_source (name, source_type, config, status) VALUES
('es-main', 'elasticsearch',
 JSON_OBJECT('addresses', JSON_ARRAY('http://10.10.10.50:9200'), 'username', '', 'password', ''),
 1);

-- 结果存储（MySQL）
INSERT INTO trend_storage_config (name, source_type, config, status) VALUES
('mysql-results', 'mysql',
 JSON_OBJECT(
   'host', '10.10.10.40', 'port', 3306, 'dbname', 'trend_metrics',
   'user', 'trend_writer', 'password', '...',
   'max_idle_conns', 10, 'max_open_conns', 100,
   'conn_max_lifetime', 1, 'debug', false
 ),
 1);
```

### 5. 启动

```bash
# Master（参与 Leader 选举，只有 Leader 实际下发任务）
./bin/trend-master -config configs/config.yaml

# Worker（可启动多个）
./bin/trend-worker -config configs/config.yaml
```

## API 接口

### 趋势查询

```
GET /api/v1/trend/{task_type}
```

查询指定实例在时间窗口内的指标趋势数据，返回 Prometheus query_range 兼容格式。

**参数**：

| 参数 | 必填 | 说明 |
|------|------|------|
| `task_type` | 是 | 路径参数，如 `orzdba` |
| `metric_name` | 是 | 指标名称：`dml` / `cpu_usage` / `mem_usage` / `diskRead` / `diskWrite` / `netIn` / `netOut` |
| `calc_instance_id` | 是 | calc_instance 主键 ID，用于分表路由 |
| `p` | 是 | 分位值：`99` / `95` / `90` / `70` / `50` / `30` |
| `window_start` | 否 | RFC3339 时间，默认 1 小时前 |
| `window_end` | 否 | RFC3339 时间，默认当前时间 |

**示例**：

```bash
curl "http://localhost:8080/api/v1/trend/orzdba?metric_name=cpu_usage&calc_instance_id=1&p=99"
```

**响应**：

```json
{
  "status": "success",
  "data": {
    "resultType": "matrix",
    "result": [
      {
        "metric": {
          "calc_instance_id": "1",
          "metric_name": "cpu_usage",
          "cluster_name": "prod-cluster-1",
          "instance_name": "10.0.1.5"
        },
        "values": [
          [1714348800000, 85.2],
          [1714349100000, 88.0]
        ]
      }
    ]
  }
}
```

### 集群配置

```
GET /api/v1/config/{cluster_name}
```

Worker 启动时调用，获取集群级配置（数据源 + 存储 + etcd 地址）。

### 监控指标

| 端点 | 说明 |
|------|------|
| `GET /metrics`（Master, 默认 :8080） | `master_up`（Gauge）、`master_stale_tasks`（GaugeVec by cluster/task_type） |
| `GET /metrics`（Worker, 默认 :9090） | `worker_up`（Gauge）、`worker_task_failures_total`（CounterVec by cluster/task_type） |

## 分表说明

分位值结果按 `calc_instance_id % 10` 哈希路由到 `metric_features_00` ~ `metric_features_09` 共 10 张分表。

- `calc_instance_id` 是 `trend_orzdba_calc_instance` 表的自增主键，在创建任务时由 Master 端传入
- 查询时同样需要传入 `calc_instance_id` 作为分表路由键
- 表结构与 `trend_quantile_result` 基础表完全一致

详细表结构文档见 [docs/tables.md](docs/tables.md)。

## 开发

- 添加新任务类型：在 `internal/task` 下实现 `Task` 接口，在 `internal/config/config.go` 的发布工厂中注册
- 提交前确保 `go build ./...` 和 `go test ./...` 均通过
