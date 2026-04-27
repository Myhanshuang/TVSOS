-- 追加扩展表设计以满足一车多运和类型范围需求
-- 1. 车辆类型支持范围 (例如冷链车能运普通货物和冷链货物)
-- 我们使用一个中间表来表示车辆类型能承载的货物类型

CREATE TABLE IF NOT EXISTS `vehicle_cargo_type_mapping` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `vehicle_type` int NOT NULL COMMENT '车辆类型ID',
  `cargo_type` int NOT NULL COMMENT '货物类型ID',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_v_c_type` (`vehicle_type`, `cargo_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 2. 模拟数据初始化 (仅作示例，后续通过脚本生成更多)
-- 假设 cargo_type: 1-普通, 2-冷链, 3-危险品
-- 假设 vehicle_type: 1-普货车, 2-冷链车, 3-危险品车
-- 冷链车(2) 可以运 普通(1) 和 冷链(2)
INSERT INTO `vehicle_cargo_type_mapping` (`vehicle_type`, `cargo_type`) VALUES 
(1, 1), -- 普货车运普货
(2, 1), -- 冷链车运普货
(2, 2), -- 冷链车运冷链
(3, 3); -- 危险品车运危险品

-- 4. 车辆预占载重（用于已分配未装货的任务容量约束）
ALTER TABLE `vehicle`
ADD COLUMN `reserved_size` int NOT NULL DEFAULT 0 COMMENT '预占载重(kg)';

-- 5. 车辆详情面板字段
ALTER TABLE `vehicle`
ADD COLUMN `license` varchar(64) NOT NULL DEFAULT '' COMMENT '车牌号';

ALTER TABLE `vehicle`
ADD COLUMN `length` double NOT NULL DEFAULT 0 COMMENT '车厢长度(米)';

ALTER TABLE `vehicle`
ADD COLUMN `width` double NOT NULL DEFAULT 0 COMMENT '车厢宽度(米)';

ALTER TABLE `vehicle`
ADD COLUMN `height` double NOT NULL DEFAULT 0 COMMENT '车厢高度(米)';

ALTER TABLE `vehicle`
ADD COLUMN `wait_time` double NOT NULL DEFAULT 0 COMMENT '当前等待时长(分钟)';

ALTER TABLE `vehicle`
ADD COLUMN `total_wait_time` double NOT NULL DEFAULT 0 COMMENT '累计等待时长(分钟)';

ALTER TABLE `vehicle`
ADD COLUMN `empty_mileage` double NOT NULL DEFAULT 0 COMMENT '空驶里程(千米)';

ALTER TABLE `vehicle`
ADD COLUMN `distance` double NOT NULL DEFAULT 0 COMMENT '运输里程(千米)';

ALTER TABLE `vehicle`
ADD COLUMN `duration` double NOT NULL DEFAULT 0 COMMENT '运输时长(小时)';

ALTER TABLE `vehicle`
ADD COLUMN `angle` double NOT NULL DEFAULT 90 COMMENT '车辆角度(度)';

-- 3. 由于 shipment 到 order_task 是 1:1，一车多运实际上是多个 order_task 绑定到同一个 vehicle_id
-- 现有的 order_task 结构基本够用，但为了更好的“一车多运”管理，
-- 我们可能需要一个字段记录任务在车辆路径中的顺序，已有的 sequential 似乎是表示“任务阶段”。
-- 所以现有结构基本满足一车多运的逻辑关系，重点在于算法如何分配和执行。
