-- 添加 angle 字段用于车辆朝向
ALTER TABLE vehicle ADD COLUMN angle DOUBLE DEFAULT 0 COMMENT '车辆角度(0-360)';
