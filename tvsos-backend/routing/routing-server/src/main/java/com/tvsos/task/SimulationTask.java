package com.tvsos.task;

import com.tvsos.manager.VehicleRouteManager;
import com.tvsos.mapper.VehicleMapper;
import com.tvsos.service.VehicleService;
import entity.Vehicle;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

import java.time.LocalDateTime;
import java.util.Map;

/**
 * 仿真驱动任务
 * 负责根据路径和倍速，实时更新所有活跃车辆的位置
 */
@Slf4j
@Component
public class SimulationTask {

    @Autowired
    private VehicleRouteManager vehicleRouteManager;
    @Autowired
    private VehicleMapper vehicleMapper;
    @Autowired
    private VehicleService vehicleService;

    // 基础速度：每秒走多少个路径点
    // 假设高德路径点平均间距 10-15米，60km/h ≈ 16m/s，大约每秒走 1-2 个点。
    private static final double BASE_POINTS_PER_SECOND = 1.5;

    @Scheduled(fixedRate = 1000) // 每秒执行
    public void tick() {
        Map<Long, VehicleRouteManager.RouteState> activeRoutes = vehicleRouteManager.getAllActiveRoutes();
        if (activeRoutes.isEmpty()) {
            return;
        }

        double globalMultiplier = vehicleRouteManager.getGlobalSpeedMultiplier();
        // 实际步长 = 基础步长 * 倍速
        double step = BASE_POINTS_PER_SECOND * globalMultiplier;

        activeRoutes.forEach((vehicleId, state) -> {
            try {
                processVehicle(vehicleId, state, step);
            } catch (Exception e) {
                log.error("Simulation error for vehicle {}", vehicleId, e);
            }
        });
    }

    private void processVehicle(Long vehicleId, VehicleRouteManager.RouteState state, double step) {
        // 1. 更新内部索引
        state.setAccumulator(state.getAccumulator() + step);
        int advance = (int) state.getAccumulator();
        if (advance > 0) {
            state.setCurrentIndex(state.getCurrentIndex() + advance);
            state.setAccumulator(state.getAccumulator() - advance);
        }

        // 2. 检查是否到达终点
        boolean finished = false;
        if (state.getCurrentIndex() >= state.getPoints().size() - 1) {
            state.setCurrentIndex(state.getPoints().size() - 1); // 停在最后一点
            finished = true;
        }

        // 3. 获取当前位置
        Double[] currentPoint = state.getCurrentPoint();
        Double[] nextPoint = state.getNextPoint();
        
        if (currentPoint != null) {
            Vehicle vehicle = new Vehicle();
            vehicle.setId(vehicleId);
            vehicle.setLon(currentPoint[0]);
            vehicle.setLat(currentPoint[1]);
            vehicle.setUpdateTime(LocalDateTime.now());

            // 计算角度 (如果有下一个点)
            // 如果到达终点，保持上一个角度（不更新 angle 字段，或者沿用最后的方向）
            if (nextPoint != null && !finished) {
                double angle = calculateBearing(currentPoint[0], currentPoint[1], nextPoint[0], nextPoint[1]);
                vehicle.setAngle(angle);
            }

            // 更新数据库位置
            vehicleMapper.update(vehicle);

            // 4. 如果到达终点，触发业务流转
            if (finished) {
                log.info("Vehicle {} arrived at segment end. Triggering update.", vehicleId);
                // 必须先移除，防止重复触发
                vehicleRouteManager.removeVehicle(vehicleId);
                // 调用业务服务进行状态流转 (例如 接单->装货, 运货->卸货)
                // 重新查出来完整的 vehicle 对象以确保数据完整性 (updateVehicle 内部会查)
                Vehicle fullVehicle = vehicleMapper.getById(vehicleId);
                if (fullVehicle != null) {
                    vehicleService.updateVehicle(fullVehicle);
                }
            }
        }
    }

    /**
     * 计算两点间的方位角 (0-360, 正北为0)
     */
    private double calculateBearing(double lon1, double lat1, double lon2, double lat2) {
        double l1 = Math.toRadians(lon1);
        double l2 = Math.toRadians(lon2);
        double phi1 = Math.toRadians(lat1);
        double phi2 = Math.toRadians(lat2);
        double y = Math.sin(l2 - l1) * Math.cos(phi2);
        double x = Math.cos(phi1) * Math.sin(phi2) - Math.sin(phi1) * Math.cos(phi2) * Math.cos(l2 - l1);
        double bearing = Math.toDegrees(Math.atan2(y, x));
        return (bearing + 360) % 360;
    }
}
