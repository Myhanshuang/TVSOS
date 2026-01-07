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

    /**
     * 设置全局倍速
     */
    public void setGlobalSpeedMultiplier(double multiplier) {
        if (multiplier > 0) {
            this.globalSpeedMultiplier = multiplier;
            log.info("Global simulation speed set to: {}", multiplier);
        }
    }

    public double getGlobalSpeedMultiplier() {
        return globalSpeedMultiplier;
    }

    /**
     * 添加或更新活跃车辆的路径
     * 当车辆开始一段新的行程时调用
     *
     * @param vehicleId 车辆ID
     * @param points    完整路径点 [lon, lat]
     * @param currentLat 初始纬度 (可选，用于校验)
     * @param currentLon 初始经度 (可选，用于校验)
     */
    public void startRoute(Long vehicleId, List<Double[]> points) {
        if (points == null || points.isEmpty()) {
            log.warn("Cannot start route for vehicle {}: empty points", vehicleId);
            return;
        }
        RouteState state = new RouteState();
        state.setVehicleId(vehicleId);
        state.setPoints(points);
        state.setCurrentIndex(0); // 从第一个点开始
        state.setAccumulator(0.0);
        state.setFinished(false);
        
        activeRoutes.put(vehicleId, state);
        log.info("Vehicle {} started new route with {} points.", vehicleId, points.size());
    }

    /**
     * 移除车辆路径 (例如车辆到达终点或任务取消)
     */
    public void removeVehicle(Long vehicleId) {
        activeRoutes.remove(vehicleId);
    }

    /**
     * 获取指定车辆的当前状态
     */
    public RouteState getVehicleState(Long vehicleId) {
        return activeRoutes.get(vehicleId);
    }

    /**
     * 获取所有活跃车辆的状态
     */
    public Map<Long, RouteState> getAllActiveRoutes() {
        return activeRoutes;
    }

    /**
     * 获取指定车辆的行驶进度 (0.0 - 1.0)
     * 如果车辆不在活跃列表中，返回 0.0
     */
    public double getProgress(Long vehicleId) {
        RouteState state = activeRoutes.get(vehicleId);
        if (state == null || state.getPoints() == null || state.getPoints().isEmpty()) {
            return 0.0;
        }
        // 简单计算：当前索引 / 总点数
        // 更精确的计算需要累加距离，但对于前端展示，点数比例足够了
        double progress = (double) state.getCurrentIndex() / (state.getPoints().size() - 1);
        return Math.min(1.0, Math.max(0.0, progress));
    }

    /**
     * 内部状态类
     */
    @Data
    public static class RouteState {
        private Long vehicleId;
        private List<Double[]> points; // 路径点序列 [lon, lat]
        private int currentIndex;      // 当前在第几个点
        private double accumulator;    // 累加器，用于处理非整数步长
        private boolean finished;      // 是否已到达终点

        /**
         * 获取当前坐标
         * @return [lon, lat]
         */
        public Double[] getCurrentPoint() {
            if (points == null || points.isEmpty()) return null;
            if (currentIndex >= points.size()) return points.get(points.size() - 1);
            return points.get(currentIndex);
        }
        
        /**
         * 获取下一个坐标（用于计算角度）
         * 如果已经是最后一个点，则返回 null 或当前点
         */
        public Double[] getNextPoint() {
            if (points == null || points.isEmpty()) return null;
            if (currentIndex + 1 >= points.size()) return points.get(points.size() - 1);
            return points.get(currentIndex + 1);
        }
    }
}
