# 数据库表结构文档

> 本文档记录 Trend 项目使用的所有 MySQL 表结构。表由 GORM AutoMigrate 自动创建/维护，也可手动执行以下 SQL 语句创建。

---

## 1. trend_cluster_task

**用途**: 集群任务类型配置表。Master 端 Scheduler 按 cron 周期读取此表，发现当前集群启用的任务类型，并据此创建 Publisher 下发任务。

| 字段 | 类型 | 约束 | 默认值 | 说明 |
|------|------|------|--------|------|
| id | bigint unsigned | PK, AUTO_INCREMENT | — | 主键 ID |
| cluster_name | varchar(64) | NOT NULL | — | 集群名称 |
| task_name | varchar(64) | NOT NULL | — | 任务类型名称，如 `orzdba` |
| status | tinyint | NOT NULL | 0 | 任务状态：0-禁用，1-启用 |
| slide_interval | int unsigned | NOT NULL | 0 | 滑动窗口大小（分钟），用于计算分位值的时间范围 |
| stale_threshold_minutes | int | NOT NULL | 30 | 超时阈值（分钟），用于判断 calc_instance 是否长时间未处理 |
| create_time | datetime | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| update_time | datetime | NOT NULL | CURRENT_TIMESTAMP ON UPDATE | 修改时间 |

```sql
CREATE TABLE trend_cluster_task (
    id                      BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    cluster_name            VARCHAR(64)  NOT NULL COMMENT '集群名称',
    task_name               VARCHAR(64)  NOT NULL COMMENT '任务类型名称',
    status                  TINYINT      NOT NULL DEFAULT 0 COMMENT '状态：0-禁用，1-启用',
    slide_interval          INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '滑动窗口大小（分钟）',
    stale_threshold_minutes INT          NOT NULL DEFAULT 30 COMMENT '超时阈值（分钟）',
    create_time             DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    update_time             DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    UNIQUE INDEX uk_cluster_task (cluster_name, task_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='集群任务类型配置表';
```

**查询方式**:
```sql
SELECT * FROM trend_cluster_task WHERE cluster_name = ? AND status = 1;
```

---

## 2. trend_orzdba_calc_instance

**用途**: orzdba 任务的实例级子任务配置表。记录哪些数据库实例需要参与分位值计算，Master 端 OrzdbaPublisher 初始化时从此表读取实例列表，为每个实例生成一个 Task 对象下发到 etcd。

| 字段 | 类型 | 约束 | 默认值 | 说明 |
|------|------|------|--------|------|
| id | bigint unsigned | PK, AUTO_INCREMENT | — | 主键 ID |
| cluster_name | varchar(64) | — | — | 集群名称 |
| instance_name | varchar(64) | NOT NULL | — | 数据库实例名称（主机名/IP），同时用于 ES 查询的 `ip` 字段匹配 |
| status | tinyint | NOT NULL | 0 | 实例状态：0-禁用，1-启用 |
| last_time | datetime | NOT NULL | CURRENT_TIMESTAMP | 上次处理完成的时间窗口终点（滑动窗口水位线） |
| create_time | datetime | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| update_time | datetime | NOT NULL | CURRENT_TIMESTAMP ON UPDATE | 修改时间 |

```sql
CREATE TABLE trend_orzdba_calc_instance (
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    cluster_name  VARCHAR(64) DEFAULT NULL COMMENT '集群名称',
    instance_name VARCHAR(64) NOT NULL COMMENT '数据库实例名称',
    status        TINYINT     NOT NULL DEFAULT 0 COMMENT '状态：0-禁用，1-启用',
    last_time     DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '上次处理完成的时间窗口终点',
    create_time   DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    update_time   DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    UNIQUE INDEX uk_cluster_instance (cluster_name, instance_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='orzdba任务实例级子任务配置表';
```

**查询方式**:
```sql
SELECT * FROM trend_orzdba_calc_instance WHERE cluster_name = ? AND status = 1;
```

---

## 3. trend_quantile_result

**用途**: 分位值计算结果表（基础表结构）。Worker 端执行完 orzdba 任务后，将各维度指标的分位值写入分表 `metric_features_00` ~ `metric_features_09`，实际表结构与本表一致。

| 字段 | 类型 | 约束 | 默认值 | 说明 |
|------|------|------|--------|------|
| id | bigint | PK, AUTO_INCREMENT | — | 主键 ID |
| cluster_name | varchar(128) | NOT NULL, INDEX | — | 集群名称 |
| task_id | varchar(128) | NOT NULL | — | 任务 ID |
| host | varchar(128) | NOT NULL | — | 数据库实例主机名 |
| metric_name | varchar(64) | NOT NULL | — | 指标名称 |
| p99 | double | NOT NULL | — | 99 分位值 |
| p95 | double | NOT NULL | — | 95 分位值 |
| p90 | double | NOT NULL | — | 90 分位值 |
| p70 | double | NOT NULL | — | 70 分位值 |
| p50 | double | NOT NULL | — | 50 分位值 |
| p30 | double | NOT NULL | — | 30 分位值 |
| sample_count | int | NOT NULL | — | 采样数据点数量 |
| window_start | datetime | NOT NULL | — | 滑动窗口起始时间 |
| window_end | datetime | NOT NULL | — | 滑动窗口结束时间 |
| created_at | datetime | — | AUTO CREATE | 记录写入时间 |

```sql
CREATE TABLE trend_quantile_result (
    id           BIGINT         AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    cluster_name VARCHAR(128)   NOT NULL COMMENT '集群名称',
    task_id      VARCHAR(128)   NOT NULL COMMENT '任务ID',
    host         VARCHAR(128)   NOT NULL COMMENT '数据库实例主机名',
    metric_name  VARCHAR(64)    NOT NULL COMMENT '指标名称',
    p99          DOUBLE         NOT NULL COMMENT '99分位值',
    p95          DOUBLE         NOT NULL COMMENT '95分位值',
    p90          DOUBLE         NOT NULL COMMENT '90分位值',
    p70          DOUBLE         NOT NULL COMMENT '70分位值',
    p50          DOUBLE         NOT NULL COMMENT '50分位值',
    p30          DOUBLE         NOT NULL COMMENT '30分位值',
    sample_count INT            NOT NULL COMMENT '采样数据点数量',
    window_start DATETIME       NOT NULL COMMENT '滑动窗口起始时间',
    window_end   DATETIME       NOT NULL COMMENT '滑动窗口结束时间',
    created_at   DATETIME       DEFAULT CURRENT_TIMESTAMP COMMENT '记录写入时间',
    INDEX idx_cluster_name (cluster_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='分位值计算结果表（基础结构）';
```

**分表策略**: `calc_instance_id % 10`，实际写入 `metric_features_00` ~ `metric_features_09`。`calc_instance_id` 取自 `trend_orzdba_calc_instance` 表的自增主键。

---

## 4. metric_features_00 ~ metric_features_09

**用途**: `trend_quantile_result` 的实际分表存储。表结构与 `trend_quantile_result` 完全一致，以下 SQL 以 `metric_features_00` 为例，其余 9 张表只需替换表名即可。

```sql
CREATE TABLE metric_features_00 (
    id           BIGINT         AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    cluster_name VARCHAR(128)   NOT NULL COMMENT '集群名称',
    task_id      VARCHAR(128)   NOT NULL COMMENT '任务ID',
    host         VARCHAR(128)   NOT NULL COMMENT '数据库实例主机名',
    metric_name  VARCHAR(64)    NOT NULL COMMENT '指标名称',
    p99          DOUBLE         NOT NULL COMMENT '99分位值',
    p95          DOUBLE         NOT NULL COMMENT '95分位值',
    p90          DOUBLE         NOT NULL COMMENT '90分位值',
    p70          DOUBLE         NOT NULL COMMENT '70分位值',
    p50          DOUBLE         NOT NULL COMMENT '50分位值',
    p30          DOUBLE         NOT NULL COMMENT '30分位值',
    sample_count INT            NOT NULL COMMENT '采样数据点数量',
    window_start DATETIME       NOT NULL COMMENT '滑动窗口起始时间',
    window_end   DATETIME       NOT NULL COMMENT '滑动窗口结束时间',
    created_at   DATETIME       DEFAULT CURRENT_TIMESTAMP COMMENT '记录写入时间',
    INDEX idx_cluster_name (cluster_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='分位值结果分表 00';
```

> **提示**：使用以下脚本批量创建 10 张分表：
>
> ```sql
> SET @i = 0;
> WHILE @i < 10 DO
>   SET @sql = CONCAT('CREATE TABLE metric_features_0', @i, ' LIKE trend_quantile_result');
>   PREPARE stmt FROM @sql;
>   EXECUTE stmt;
>   SET @i = @i + 1;
> END WHILE;
> ```
>
> 或手动执行 10 次 `CREATE TABLE metric_features_0X LIKE trend_quantile_result;`（X = 0~9）。

---

## 5. trend_data_source

**用途**: 数据源注册表。记录 Worker 端可读取的数据源信息（Elasticsearch、MySQL、Kafka 等），连接配置以 JSON 格式存储。Worker 启动时通过 Master API 获取此表数据，初始化对应的读客户端。管理页面通过 CRUD 此表来维护数据源配置。

| 字段 | 类型 | 约束 | 默认值 | 说明 |
|------|------|------|--------|------|
| id | bigint unsigned | PK, AUTO_INCREMENT | — | 主键 ID |
| name | varchar(64) | UNIQUE, NOT NULL | — | 数据源名称 |
| source_type | varchar(32) | NOT NULL | — | 数据源类型：`elasticsearch` / `mysql` / `kafka` |
| config | json | NOT NULL | — | 连接配置，JSON 格式 |
| status | tinyint | NOT NULL | 1 | 数据源状态：0-禁用，1-启用 |
| create_time | datetime | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |

```sql
CREATE TABLE trend_data_source (
    id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    name        VARCHAR(64) NOT NULL COMMENT '数据源名称',
    source_type VARCHAR(32) NOT NULL COMMENT '数据源类型',
    config      JSON        NOT NULL COMMENT '连接配置(JSON)',
    status      TINYINT     NOT NULL DEFAULT 1 COMMENT '状态：0-禁用，1-启用',
    create_time DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    UNIQUE INDEX uk_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='数据源注册表';
```

**config JSON 示例**:

Elasticsearch:
```json
{
  "addresses": ["http://es1:9200", "http://es2:9200"],
  "username": "trend_reader",
  "password": "..."
}
```

Kafka:
```json
{
  "brokers": ["kafka1:9092", "kafka2:9092"],
  "group_id": "trend-workers",
  "username": "",
  "password": ""
}
```

MySQL（只读数据源）:
```json
{
  "host": "mysql-ro",
  "port": 3306,
  "dbname": "source_db",
  "user": "reader",
  "password": "..."
}
```

---

## 6. trend_storage_config

**用途**: 存储配置表。记录 Worker 端写入分位值结果的目标存储（MySQL、Elasticsearch 等），连接配置以 JSON 格式存储。Worker 启动时通过 Master API 获取此表数据，初始化对应的写入客户端。数组结构支持多个存储目标。

| 字段 | 类型 | 约束 | 默认值 | 说明 |
|------|------|------|--------|------|
| id | bigint unsigned | PK, AUTO_INCREMENT | — | 主键 ID |
| name | varchar(64) | UNIQUE, NOT NULL | — | 存储名称 |
| source_type | varchar(32) | NOT NULL | — | 存储类型：`mysql` / `elasticsearch` |
| config | json | NOT NULL | — | 连接配置，JSON 格式 |
| status | tinyint | NOT NULL | 1 | 存储状态：0-禁用，1-启用 |
| create_time | datetime | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |

```sql
CREATE TABLE trend_storage_config (
    id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    name        VARCHAR(64) NOT NULL COMMENT '存储名称',
    source_type VARCHAR(32) NOT NULL COMMENT '存储类型',
    config      JSON        NOT NULL COMMENT '连接配置(JSON)',
    status      TINYINT     NOT NULL DEFAULT 1 COMMENT '状态：0-禁用，1-启用',
    create_time DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    UNIQUE INDEX uk_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='存储配置表';
```

**config JSON 示例**:

MySQL:
```json
{
  "host": "mysql-prod",
  "port": 3306,
  "dbname": "trend_metrics",
  "user": "trend_writer",
  "password": "...",
  "max_idle_conns": 10,
  "max_open_conns": 100,
  "conn_max_lifetime": 1,
  "debug": false
}
```

---

## 初始化数据示例

项目部署后可向 `trend_data_source` 和 `trend_storage_config` 插入初始配置：

```sql
-- 注册 Elasticsearch 数据源
INSERT INTO trend_data_source (name, source_type, config, status) VALUES
('es-main', 'elasticsearch',
 JSON_OBJECT('addresses', JSON_ARRAY('http://10.10.10.50:9200'), 'username', '', 'password', ''),
 1);

-- 注册 MySQL 结果存储
INSERT INTO trend_storage_config (name, source_type, config, status) VALUES
('mysql-results', 'mysql',
 JSON_OBJECT(
   'host', '10.10.10.40', 'port', 3306, 'dbname', 'trend_metrics',
   'user', 'trend_writer', 'password', '...',
   'max_idle_conns', 10, 'max_open_conns', 100,
   'conn_max_lifetime', 1, 'debug', false
 ),
 1);

-- 注册 orzdba 任务类型
INSERT INTO trend_cluster_task (cluster_name, task_name, status, slide_interval) VALUES
('prod-cluster-1', 'orzdba', 1, 5);

-- 注册需要监控的数据库实例
INSERT INTO trend_orzdba_calc_instance (cluster_name, instance_name, status, last_time) VALUES
('prod-cluster-1', '10.0.1.5', 1, NOW()),
('prod-cluster-1', '10.0.1.6', 1, NOW());
```

---

## 表间关系

```
trend_cluster_task (任务类型配置)
       │
       │ task_name → 匹配 Publisher 类型
       ▼
trend_orzdba_calc_instance (实例级子任务)
       │
       │ 每个实例生成一个 Task → 下发到 etcd
       ▼
Worker 从 etcd 订阅 → 执行 ES 查询 → 计算分位值
       │
       ▼
metric_features_00~09 (分位值结果分表)
       ▲
       │ Worker 根据 trend_storage_config 初始化写入连接
       │ Worker 根据 trend_data_source 初始化数据源连接

trend_data_source (数据源注册) ──→ Worker 读取 ES/Kafka/MySQL
trend_storage_config (存储注册) ──→ Worker 写入 MySQL/ES
```
