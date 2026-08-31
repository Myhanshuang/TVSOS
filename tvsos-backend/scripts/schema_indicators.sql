-- ============================================================================
-- TVSOS 科学指标体系 数据库设计说明
-- 依据《plans/科学指标体系报告.md》§3.5 数据补齐方案落地
-- 说明：这些表/字段已由 internal/database/db.go 的启动迁移自动创建（CREATE TABLE IF
-- NOT EXISTS / ALTER TABLE 加列），本脚本仅作为 DBA 手工执行与文档存档备查。
-- ============================================================================

-- 1) metric_snapshot 指标历史采样表（P1）：支撑历史趋势、成本效益对比(C1)、可用性(R3)
CREATE TABLE IF NOT EXISTS `metric_snapshot` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `snapshot_time` datetime NOT NULL,
  `cost_value` double NOT NULL DEFAULT 0 COMMENT '综合成本F',
  `total_throughput` double NOT NULL DEFAULT 0 COMMENT '总吞吐量(吨)',
  `cargo_pending` double NOT NULL DEFAULT 0 COMMENT '待处理货物(吨)',
  `transport_efficiency` double NOT NULL DEFAULT 0 COMMENT '运输效率(载重利用率)',
  `total_waiting_time` double NOT NULL DEFAULT 0 COMMENT '车辆总等待(秒)',
  `total_empty_mileage` double NOT NULL DEFAULT 0 COMMENT '总空驶里程(米)',
  `unit_cost` double NOT NULL DEFAULT 0 COMMENT '单位成本(C2)',
  `empty_rate` double NOT NULL DEFAULT 0 COMMENT '空驶率(U4)',
  `loading_rate_stddev` double NOT NULL DEFAULT 0 COMMENT '装载率标准差(U2)',
  `match_response_p50` double NOT NULL DEFAULT 0 COMMENT '匹配响应P50(T2)',
  `match_response_p95` double NOT NULL DEFAULT 0 COMMENT '匹配响应P95(T2)',
  `match_response_p99` double NOT NULL DEFAULT 0 COMMENT '匹配响应P99(T2)',
  `gini_coefficient` double NOT NULL DEFAULT 0 COMMENT '基尼系数(F1)',
  `carbon_total` double NOT NULL DEFAULT 0 COMMENT '总碳排放kg(G1)',
  `carbon_per_ton_km` double NOT NULL DEFAULT 0 COMMENT '单位吨公里碳排放(G2)',
  PRIMARY KEY (`id`),
  KEY `idx_snapshot_time` (`snapshot_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 2) system_event 系统运行事件表（P1）：支撑可靠性指标 MTBF/MTTR/可用性(R1-R3)
CREATE TABLE IF NOT EXISTS `system_event` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `event_time` datetime NOT NULL,
  `event_type` varchar(32) NOT NULL DEFAULT '' COMMENT 'startup/recovery/fault/recover',
  `level` varchar(16) NOT NULL DEFAULT 'info' COMMENT '级别',
  `message` varchar(512) NOT NULL DEFAULT '' COMMENT '描述',
  `detail` text COMMENT '详情(JSON)',
  PRIMARY KEY (`id`),
  KEY `idx_event_time` (`event_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 3) emission_factor 排放因子配置表（P1）：绿色指标 G1/G2 的配置数据源
CREATE TABLE IF NOT EXISTS `emission_factor` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `vehicle_type` int NOT NULL COMMENT '车辆类型(对应vehicle.tybe)',
  `factor_empty` double NOT NULL DEFAULT 0 COMMENT '空驶排放因子(kg/km)',
  `factor_load` double NOT NULL DEFAULT 0 COMMENT '载重排放因子(kg/km)',
  `remark` varchar(255) NOT NULL DEFAULT '' COMMENT '备注',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_vehicle_type` (`vehicle_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 4) handling_config 装卸/调度参数配置表（P1）：将经验系数(如装卸60秒/吨)配置化
CREATE TABLE IF NOT EXISTS `handling_config` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `config_key` varchar(64) NOT NULL COMMENT '配置键',
  `config_value` double NOT NULL DEFAULT 0 COMMENT '配置值',
  `remark` varchar(255) NOT NULL DEFAULT '' COMMENT '说明',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_config_key` (`config_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 5) matching_log 匹配决策日志表（P2）：匹配精准度指标(M1-M4, MRR)的评估数据源
CREATE TABLE IF NOT EXISTS `matching_log` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `shipment_id` bigint unsigned NOT NULL COMMENT '运单id',
  `chosen_vehicle_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '选中车辆id',
  `candidate_ranking` text COMMENT '候选车辆排名(JSON)',
  `ground_truth_vehicle` bigint unsigned NOT NULL DEFAULT 0 COMMENT '最优车辆(离线回放评估用)',
  `created_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_shipment` (`shipment_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 6) track_log 车辆轨迹/里程日志表（P2）：里程利用率(U1)与碳排放(G1)的真实化数据源
CREATE TABLE IF NOT EXISTS `track_log` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `vehicle_id` bigint unsigned NOT NULL COMMENT '车辆id',
  `lon` double NOT NULL DEFAULT 0 COMMENT '经度',
  `lat` double NOT NULL DEFAULT 0 COMMENT '纬度',
  `is_empty` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否空驶段',
  `record_time` datetime NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_vehicle_time` (`vehicle_id`, `record_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 7) order_task 任务阶段时间戳（P1）：还原调度时序，支撑 T2 响应时间分位/瓶颈分析/装卸时间标定
ALTER TABLE `order_task` ADD COLUMN IF NOT EXISTS `assign_time` datetime DEFAULT NULL COMMENT '订单被分配车辆时间';
ALTER TABLE `order_task` ADD COLUMN IF NOT EXISTS `reach_pickup_time` datetime DEFAULT NULL COMMENT '车辆到达提货点时间';
ALTER TABLE `order_task` ADD COLUMN IF NOT EXISTS `load_start_time` datetime DEFAULT NULL COMMENT '开始装货时间';
ALTER TABLE `order_task` ADD COLUMN IF NOT EXISTS `load_end_time` datetime DEFAULT NULL COMMENT '完成装货时间';
ALTER TABLE `order_task` ADD COLUMN IF NOT EXISTS `transport_start_time` datetime DEFAULT NULL COMMENT '开始运输时间';
ALTER TABLE `order_task` ADD COLUMN IF NOT EXISTS `reach_delivery_time` datetime DEFAULT NULL COMMENT '到达收货点时间';
ALTER TABLE `order_task` ADD COLUMN IF NOT EXISTS `unload_start_time` datetime DEFAULT NULL COMMENT '开始卸货时间';
ALTER TABLE `order_task` ADD COLUMN IF NOT EXISTS `unload_end_time` datetime DEFAULT NULL COMMENT '完成卸货时间';
ALTER TABLE `order_task` ADD COLUMN IF NOT EXISTS `signoff_time` datetime DEFAULT NULL COMMENT '签收完成时间';

-- 可选：预置默认排放因子示例（按需启用）
-- INSERT INTO `emission_factor` (`vehicle_type`, `factor_empty`, `factor_load`, `remark`) VALUES
--   (1, 0.60, 1.10, '普通厢式货车'),
--   (2, 0.80, 1.50, '冷藏/重型货车');