-- Trend 数据库表结构
-- 表由 GORM AutoMigrate 自动创建，也可手动执行此文件初始化

SET NAMES utf8mb4;

-- ============================================================
-- 1. 集群任务类型配置表
-- ============================================================
CREATE TABLE IF NOT EXISTS trend_cluster_task (
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

-- ============================================================
-- 2. orzdba 任务实例级子任务配置表
-- ============================================================
CREATE TABLE IF NOT EXISTS trend_orzdba_calc_instance (
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    cluster_name  VARCHAR(64) DEFAULT NULL COMMENT '集群名称',
    instance_name VARCHAR(64) NOT NULL COMMENT '数据库实例名称',
    status        TINYINT     NOT NULL DEFAULT 0 COMMENT '状态：0-禁用，1-启用',
    last_time     DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '上次处理完成的时间窗口终点',
    create_time   DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    update_time   DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    UNIQUE INDEX uk_cluster_instance (cluster_name, instance_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='orzdba任务实例级子任务配置表';

-- ============================================================
-- 3. 分位值计算结果表（基础结构，实际数据写入 metric_features_xx 分表）
-- ============================================================
CREATE TABLE IF NOT EXISTS trend_quantile_result (
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

-- ============================================================
-- 4. 分位值结果分表（按 calc_instance_id % 10 路由）
-- ============================================================
CREATE TABLE IF NOT EXISTS metric_features_00 (
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

CREATE TABLE IF NOT EXISTS metric_features_01 LIKE metric_features_00;
ALTER TABLE metric_features_01 COMMENT='分位值结果分表 01';

CREATE TABLE IF NOT EXISTS metric_features_02 LIKE metric_features_00;
ALTER TABLE metric_features_02 COMMENT='分位值结果分表 02';

CREATE TABLE IF NOT EXISTS metric_features_03 LIKE metric_features_00;
ALTER TABLE metric_features_03 COMMENT='分位值结果分表 03';

CREATE TABLE IF NOT EXISTS metric_features_04 LIKE metric_features_00;
ALTER TABLE metric_features_04 COMMENT='分位值结果分表 04';

CREATE TABLE IF NOT EXISTS metric_features_05 LIKE metric_features_00;
ALTER TABLE metric_features_05 COMMENT='分位值结果分表 05';

CREATE TABLE IF NOT EXISTS metric_features_06 LIKE metric_features_00;
ALTER TABLE metric_features_06 COMMENT='分位值结果分表 06';

CREATE TABLE IF NOT EXISTS metric_features_07 LIKE metric_features_00;
ALTER TABLE metric_features_07 COMMENT='分位值结果分表 07';

CREATE TABLE IF NOT EXISTS metric_features_08 LIKE metric_features_00;
ALTER TABLE metric_features_08 COMMENT='分位值结果分表 08';

CREATE TABLE IF NOT EXISTS metric_features_09 LIKE metric_features_00;
ALTER TABLE metric_features_09 COMMENT='分位值结果分表 09';

-- ============================================================
-- 5. 数据源注册表
-- ============================================================
CREATE TABLE IF NOT EXISTS trend_data_source (
    id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    name        VARCHAR(64) NOT NULL COMMENT '数据源名称',
    source_type VARCHAR(32) NOT NULL COMMENT '数据源类型：elasticsearch/mysql/kafka',
    config      JSON        NOT NULL COMMENT '连接配置(JSON)',
    status      TINYINT     NOT NULL DEFAULT 1 COMMENT '状态：0-禁用，1-启用',
    create_time DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    UNIQUE INDEX uk_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='数据源注册表';

-- ============================================================
-- 6. 存储配置表
-- ============================================================
CREATE TABLE IF NOT EXISTS trend_storage_config (
    id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
    name        VARCHAR(64) NOT NULL COMMENT '存储名称',
    source_type VARCHAR(32) NOT NULL COMMENT '存储类型：mysql/elasticsearch',
    config      JSON        NOT NULL COMMENT '连接配置(JSON)',
    status      TINYINT     NOT NULL DEFAULT 1 COMMENT '状态：0-禁用，1-启用',
    create_time DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    UNIQUE INDEX uk_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='存储配置表';
