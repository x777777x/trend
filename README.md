# Trend 数据统计分析平台

Trend 是一个基于 Go 构建的分布式任务调度与数据统计分析平台，主要用于对业务系统的指标数据（如数据库指标等）进行定时获取、清洗和聚合计算（例如 p99、p95、p90 等分位值分析），从而为系统运维和性能优化提供数据支撑。

## 🎯 项目功能

- **分布式任务调度**：基于 Etcd 进行 Master 节点选举（Leader Election），保证高可用性，同时利用 gocron 实现灵活的定时任务调度。
- **配置与任务管理**：通过 MySQL 维护集群和各任务的配置项，动态加载启用的任务清单。
- **任务分发与执行**：Master 节点将生成的任务分发给 Worker 节点消费执行。
- **数据提取与计算**：Worker 节点根据任务指令从 Elasticsearch（ES）中拉取原始监控记录，计算特定维度指标（DML、CPU使用率、内存使用率、磁盘IO等）的分位值，并最终归档。
- **弹性扩展**：Master 和 Worker 服务解耦，可按需启动多个 Worker 实例来提升处理能力。

## 🏗 代码结构

项目采用典型的 Go 项目结构 `Standard Go Project Layout` 进行组织：

```text
trend/
├── cmd/
│   ├── master/      # Master 节点启动入口
│   └── worker/      # Worker 节点启动入口
├── configs/         # 配置文件存储目录及示例
├── docs/            # 项目文档记录
├── internal/        # 核心业务逻辑实现 (禁止外部模块依赖)
│   ├── config/      # Viper 配置解析与映射
│   ├── master/      # Master 端调度器 (Scheduler)、发布者 (Publisher)、Leader 选举实现
│   ├── models/      # 数据库模型 (GORM Object Mapping)
│   ├── task/        # 具体的任务抽象及实现 (如 orzdba 指标聚合任务)
│   └── worker/      # Worker 端消费者 (Consumer)、执行器 (Executor)
├── pkg/             # 公共组件库、第三方中间件封装
│   ├── algo/        # 统计算法 (如分位值计算)
│   ├── etcd/        # Etcd 连接与分布式锁封装
│   ├── logger/      # 日志记录模块 (基于 Uber Zap)
│   ├── storage/     # ES 查询和 MySQL(GORM) 存储连接池
│   └── utils/       # 常用工具函数
├── go.mod           # Go 依赖管理
└── main.exe         # Windows 下的编译产物示例
```

## 🛠 技术栈

- **Golang**: `1.25.7` 强类型并发编程语言，作为整个项目的基石。
- **数据库/存储**:
  - **MySQL**: 存储结构化的任务与集群配置，ORM 采用 [GORM](https://gorm.io/) `v1.31.1`。
  - **Elasticsearch (ES)**: 采用 [ES v8 go client](https://github.com/elastic/go-elasticsearch) 进行时序日志与监控数据的提取。
  - **Etcd**: 采用 `v3.6.8`，用于分布式环境下的节点注册发现以及 Master 集群的 Leader 选主。
- **定时调度**: [gocron/v2](https://github.com/go-co-op/gocron) 管理复杂的任务触发编排。
- **其他关键依赖**:
  - `viper` 用于灵活的配置管理。
  - `zap` 和 `lumberjack` 作为高性能分级日志和滚动切分组件。

## 🚀 快速开始

### 1. 环境准备

确保机器已安装 Go `1.25` 及以上版本环境。并准备好相应的各类中间件：
- MySQL (需预先建立好相应的数据库及表结构，例如 `trend_cluster_task` 和 `trend_orzdba_calc_instances`)
- Elasticsearch (v8)
- Etcd

### 2. 构建项目

你可以分别编译 Master 和 Worker 的可执行程序：

```bash
# 编译 master
go build -o bin/trend-master ./cmd/master/main.go

# 编译 worker
go build -o bin/trend-worker ./cmd/worker/main.go
```

### 3. 配置修改

在 `configs` 目录下准备对应的 YAML 配置文件，确保 MySQL、Etcd、ES 等连接项及 Master/Worker 特有参数正确。

### 4. 运行服务

**启动 Master 节点**
Master 负责定时查询需要执行的数据库和任务配置，通过发布机制下发到执行队列，只有竞争到 leader 身份的节点才会实际下发任务。

```bash
./bin/trend-master -config configs/config.yaml
```

**启动 Worker 节点**
可以启动多个 Worker，Worker 负责监听下发的任务并进行耗时的 ES 请求及聚合运算。

```bash
./bin/trend-worker -config configs/config.yaml
```

## 📝 贡献与维护

开发时请遵循现有的代码规范：
- 添加新任务类型请在 `internal/task` 下新增逻辑，并在 `internal/master/publisher.go` 的发布工厂函数中进行注册。
- 提供具有描述性的 commit 信息。
