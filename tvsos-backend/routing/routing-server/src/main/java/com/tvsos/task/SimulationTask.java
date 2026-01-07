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

    @Autowired
    private com.tvsos.service.RouteStorageService routeStorageService;
    @Autowired
    private com.tvsos.mapper.TripMapper tripMapper;

    // 基础速度：每秒走多少个路径点
    private static final double BASE_POINTS_PER_SECOND = 1.5;

    @Scheduled(fixedRate = 1000) // 每秒执行
    public void tick() {
        // 1. 处理移动中的车辆 (内存驱动)
        handleMovingVehicles();
        
        // [New] 2. 检查并唤醒“失活”的移动车辆 (数据库驱动)
        // 防止服务重启或逻辑遗漏导致状态为 2/4 的车辆在内存中不存在
        handleReviveVehicles();
        
        // 3. 处理静止/作业中的车辆 (数据库驱动)
        // 状态 3(装货) 和 5(卸货)
        handleStationaryVehicles();
    }

    private void handleMovingVehicles() {
        Map<Long, VehicleRouteManager.RouteState> activeRoutes = vehicleRouteManager.getAllActiveRoutes();
        // 即使为空也要继续执行下面的 revive 逻辑
        
        double globalMultiplier = vehicleRouteManager.getGlobalSpeedMultiplier();
        double step = BASE_POINTS_PER_SECOND * globalMultiplier;

        if (!activeRoutes.isEmpty()) {
            activeRoutes.forEach((vehicleId, state) -> {
                try {
                    processVehicle(vehicleId, state, step);
                } catch (Exception e) {
                    log.error("Simulation error for vehicle {}", vehicleId, e);
                }
            });
        }
    }
    
    /**
     * 唤醒机制：查找状态为 2 或 4 但不在 activeRoutes 中的车辆
     */
    private void handleReviveVehicles() {
        // 获取一批待处理车辆 (这里复用 getPendingVehicles，假设量不大)
        // 更好的做法是专门写一个 SQL: select * from vehicle where status in (2,4)
        // 这里先遍历检查
        java.util.List<Vehicle> vehicles = vehicleService.getPendingVehicles(50);
        Map<Long, VehicleRouteManager.RouteState> activeRoutes = vehicleRouteManager.getAllActiveRoutes();
        
        for (Vehicle v : vehicles) {
            if ((v.getStatus() == 2 || v.getStatus() == 4) && !activeRoutes.containsKey(v.getId())) {
                log.warn("发现失活车辆 {} (Status={})，尝试唤醒...", v.getId(), v.getStatus());
                reviveVehicle(v);
            }
        }
    }
    
    private void reviveVehicle(Vehicle v) {
        // 查找对应的 Trip
        entity.Trip trip = tripMapper.getByVehicleIdAndStatus(v.getId(), 2);
        if (trip == null) {
            log.warn("车辆 {} 唤醒中止：未找到进行中的 Trip (Status=2)。强制重置为空闲状态。", v.getId());
            // 强制重置车辆状态为 1 (空闲)，避免死循环
            v.setStatus(1);
            v.setUpdateTime(LocalDateTime.now());
            vehicleMapper.update(v);
            return;
        }
        
        int segmentIndex = (v.getStatus() == 2) ? 1 : 2;
        
        try {
            java.util.List<Double[]> points = routeStorageService.loadRoute(trip.getId(), segmentIndex);
            if (points != null && !points.isEmpty()) {
                vehicleRouteManager.startRoute(v.getId(), points);
                
                // 此时需要根据当前数据库里的位置，找到路径上最近的点作为 currentIndex
                // 否则会从头开始跑，导致“瞬移回起点”。
                // 这里简单起见，如果数据库位置有效，尝试匹配；否则从头开始。
                // TODO: 增加“最近点匹配”逻辑。目前先从头开始，保证能动。
                log.info("车辆 {} 已唤醒，重新加入仿真队列", v.getId());
            } else {
                log.error("车辆 {} 唤醒失败：无法加载路径 (Trip={}, Seg={})", v.getId(), trip.getId(), segmentIndex);
            }
        } catch (Exception e) {
            log.error("车辆 {} 唤醒异常", v.getId(), e);
        }
    }

    // 处理静止车辆的状态流转 (装货 -> 运货, 卸货 -> 空闲)
    private void handleStationaryVehicles() {
        // 查找状态为 3 或 5 的车辆
        // 为了性能，一次只处理一批，或者依靠数据库索引
        // 这里简化逻辑：调用 Service 获取需要更新的静止车辆
        // 实际上可以直接复用 vehicleMapper.getPendingVehicles 类似的逻辑，但要带状态过滤
        // 由于这里不能直接改 Mapper 接口定义（尽量少改），我们假设用 getPendingVehicles 查出来后过滤
        
        // 稍微hack一下：我们利用 vehicleService.getPendingVehicles，它按时间排序
        // 只要我们处理了，updateTime 就会更新，下次就不会查到它了
        java.util.List<Vehicle> vehicles = vehicleService.getPendingVehicles(50); 
        
        // 停留时间阈值 (秒)。倍速下，停留时间应该缩短
        double globalMultiplier = vehicleRouteManager.getGlobalSpeedMultiplier();
        long requiredStaySeconds = (long) (5.0 / globalMultiplier); // 基础5秒
        if (requiredStaySeconds < 1) requiredStaySeconds = 1;

        for (Vehicle v : vehicles) {
            int status = v.getStatus();
            if (status == 3 || status == 5) {
                // 检查停留时间
                java.time.Duration duration = java.time.Duration.between(v.getUpdateTime(), LocalDateTime.now());
                if (duration.getSeconds() >= requiredStaySeconds) {
                    log.info("车辆 {} 状态 {} 停留结束，触发流转", v.getId(), status);
                    vehicleService.updateVehicle(v);
                }
            }
        }
    }

    private void processVehicle(Long vehicleId, VehicleRouteManager.RouteState state, double step) {
        // 1. 更新内部索引
        state.setAccumulator(state.getAccumulator() + step);
        int advance = (int) state.getAccumulator();
        if (advance > 0) {
            state.setCurrentIndex(state.getCurrentIndex() + advance);
            state.setAccumulator(state.getAccumulator() - advance);
            // Log movement periodically to avoid spam (e.g. every 10th move or if state changes significantly)
            // log.debug("Vehicle {} advanced to index {}", vehicleId, state.getCurrentIndex());
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
            try {
                vehicleMapper.update(vehicle);
            } catch (Exception e) {
                log.error("Failed to update vehicle position for {}", vehicleId, e);
            }

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
        } else {
            log.warn("Vehicle {} has null position at index {}", vehicleId, state.getCurrentIndex());
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
