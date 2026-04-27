-- 车辆模拟数据 (使用 REPLACE 避免 ID 重复报错)
REPLACE INTO `vehicle` (
	`id`, `license`, `lon`, `lat`, `speed`, `update_time`, `status`, `tybe`,
	`size`, `reserved_size`, `capacity`, `length`, `width`, `height`,
	`wait_time`, `total_wait_time`, `empty_mileage`, `distance`, `duration`, `angle`
) VALUES
(1, '京A10001', 116.40, 39.90, 60.0, NOW(), 2, 1, 10, 0, 1000, 6.80, 2.40, 2.60, 12.0, 130.5, 21.4, 54.2, 1.8, 90),
(2, '京A10002', 116.41, 39.91, 55.0, NOW(), 1, 1, 10, 0, 800,  6.20, 2.30, 2.50, 0.0,  90.0, 18.2, 42.1, 1.4, 110),
(3, '京A10003', 116.42, 39.92, 0.0,  NOW(), 2, 2, 8,  0, 500,  5.90, 2.20, 2.40, 20.0, 220.0, 11.0, 33.8, 1.1, 90),
(4, '京A10004', 116.43, 39.93, 40.0, NOW(), 2, 1, 15, 0, 2000, 8.20, 2.50, 2.80, 0.0,  75.5, 27.3, 71.6, 2.3, 70),
(5, '京A10005', 116.44, 39.94, 0.0,  NOW(), 2, 1, 5,  0, 300,  5.20, 2.10, 2.20, 35.0, 300.2, 8.6,  19.4, 0.7, 90),
(6, '京A10006', 116.45, 39.95, 0.0,  NOW(), 2, 1, 10, 0, 1000, 6.60, 2.35, 2.55, 8.0,  118.9, 15.1, 47.5, 1.6, 90);

-- 货物类型模拟数据
REPLACE INTO `cargo` (`id`, `name`, `tybe`, `pack`, `weight`) VALUES
(1, 1, 1, 1, 200),  -- 普通包裹 200kg
(2, 2, 2, 2, 400),  -- 液体化学品 (危险品) 400kg
(3, 3, 1, 1, 1500); -- 大型机械 1500kg

-- 订单/运单模拟数据
REPLACE INTO `shipment` (`id`, `start_poi_id`, `end_poi_id`, `create_time`, `update_time`, `status`, `cargo_id`, `count`) VALUES
(1, 1, 2, DATE_SUB(NOW(), INTERVAL 1 HOUR), NOW(), 4, 1, 1), -- 已完成
(2, 2, 3, DATE_SUB(NOW(), INTERVAL 2 HOUR), NOW(), 3, 2, 1), -- 运输中
(3, 1, 3, NOW(), NOW(), 1, 1, 1);                            -- 待分配

-- 任务模拟数据
REPLACE INTO `order_task` (`id`, `shipment_id`, `vehicle_id`, `sequential`, `create_time`, `update_time`) VALUES
(1, 1, 1, 3, DATE_SUB(NOW(), INTERVAL 1 HOUR), DATE_SUB(NOW(), INTERVAL 20 MINUTE)), -- 已完成任务
(2, 2, 2, 2, DATE_SUB(NOW(), INTERVAL 2 HOUR), DATE_SUB(NOW(), INTERVAL 90 MINUTE)); -- 运输中任务

