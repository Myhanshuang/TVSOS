package com.tvsos.manager;

import lombok.Data;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;

import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

/**
 * 车辆路径状态管理器 (内存版)
 * 维护所有活跃车辆的实时位置索引和路径数据
 */
@Slf4j
@Component
public class VehicleRouteManager {

    // 内存存储：VehicleId -> RouteState
    private final Map<Long, RouteState> activeRoutes = new ConcurrentHashMap<>();

    // 全局倍速因子，默认为 1.0
    private volatile double globalSpeedMultiplier = 1.0;

    public void setGlobalSpeedMultiplier(double multiplier) {
        if (multiplier > 0) {
            this.globalSpeedMultiplier = multiplier;
        }
    }

    public double getGlobalSpeedMultiplier() {
        return globalSpeedMultiplier;
    }

    /**
     * 添加或更新活跃车辆的路径
     * @param durationSeconds 预计用时（秒），用于计算每秒步长
     */
    public void startRoute(Long vehicleId, List<Double[]> points, long durationSeconds) {
        if (points == null || points.isEmpty()) {
            return;
        }
        RouteState state = new RouteState();
        state.setVehicleId(vehicleId);
        state.setPoints(points);
        state.setCurrentIndex(0);
        state.setAccumulator(0.0);
        state.setFinished(false);
        state.setDurationSeconds(durationSeconds);
        
        // Calculate step (points per second)
        // If duration is 0 (teleport), step is infinite (handled in sim)
        if (durationSeconds > 0) {
            state.setPointsPerSecond((double) points.size() / durationSeconds);
        } else {
            state.setPointsPerSecond(10.0); // Default fast speed
        }
        
        activeRoutes.put(vehicleId, state);
        log.info("Vehicle {} simulation started. Points: {}, Duration: {}s, Step: {}", 
                vehicleId, points.size(), durationSeconds, state.getPointsPerSecond());
    }
    
    // Overload for backward compatibility or simple use cases
    public void startRoute(Long vehicleId, List<Double[]> points) {
        // Default to a reasonable speed if duration not provided (e.g., 60km/h approx?)
        // Or assume 1 point per second
        startRoute(vehicleId, points, points.size()); // 1 point per second default
    }

    public void removeVehicle(Long vehicleId) {
        activeRoutes.remove(vehicleId);
    }

    public RouteState getVehicleState(Long vehicleId) {
        return activeRoutes.get(vehicleId);
    }

    public Map<Long, RouteState> getAllActiveRoutes() {
        return activeRoutes;
    }

    public double getProgress(Long vehicleId) {
        RouteState state = activeRoutes.get(vehicleId);
        if (state == null || state.getPoints() == null || state.getPoints().isEmpty()) {
            return 0.0;
        }
        double progress = (double) state.getCurrentIndex() / (state.getPoints().size() - 1);
        return Math.min(1.0, Math.max(0.0, progress));
    }

    @Data
    public static class RouteState {
        private Long vehicleId;
        private List<Double[]> points;
        private int currentIndex;
        private double accumulator;
        private boolean finished;
        private long durationSeconds;
        private double pointsPerSecond; // Calculated base speed

        public Double[] getCurrentPoint() {
            if (points == null || points.isEmpty()) return null;
            if (currentIndex >= points.size()) return points.get(points.size() - 1);
            return points.get(currentIndex);
        }
        
        public Double[] getNextPoint() {
            if (points == null || points.isEmpty()) return null;
            if (currentIndex + 1 >= points.size()) return points.get(points.size() - 1);
            return points.get(currentIndex + 1);
        }
    }
}