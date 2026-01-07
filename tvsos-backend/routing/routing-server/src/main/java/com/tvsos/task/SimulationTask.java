package com.tvsos.task;

import com.tvsos.manager.VehicleRouteManager;
import com.tvsos.mapper.TripMapper;
import com.tvsos.mapper.TripSegmentMapper;
import com.tvsos.mapper.VehicleMapper;
import com.tvsos.service.RouteStorageService;
import com.tvsos.service.VehicleService;
import com.tvsos.utils.MarkovStatusUtils;
import com.tvsos.utils.TripUtils;
import entity.Trip;
import entity.TripSegment;
import entity.Vehicle;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

import java.time.LocalDateTime;
import java.util.List;
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
    private RouteStorageService routeStorageService;
    @Autowired
    private TripMapper tripMapper;
    @Autowired
    private TripUtils tripUtils;
    @Autowired
    private TripSegmentMapper tripSegmentMapper;

    // 基础速度：每秒走多少个路径点
    private static final double BASE_POINTS_PER_SECOND = 1.5;

    @Scheduled(fixedRate = 1000) // 每秒执行
    public void tick() {
        // 1. 处理移动中的车辆 (内存驱动)
        handleMovingVehicles();
        
        // 2. 检查并唤醒“失活”的移动车辆 (数据库驱动)
        handleReviveVehicles();
        
        // 3. 处理静止/作业中的车辆 (数据库驱动)
        handleStationaryVehicles();
    }

    private void handleMovingVehicles() {
        Map<Long, VehicleRouteManager.RouteState> activeRoutes = vehicleRouteManager.getAllActiveRoutes();
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
    
    private void handleReviveVehicles() {
        // 查找状态为 2 或 4 但不在 activeRoutes 中的车辆
        List<Vehicle> vehicles = vehicleService.getPendingVehicles(20);
        Map<Long, VehicleRouteManager.RouteState> activeRoutes = vehicleRouteManager.getAllActiveRoutes();
        
        for (Vehicle v : vehicles) {
            if ((v.getStatus() == 2 || v.getStatus() == 4) && !activeRoutes.containsKey(v.getId())) {
                reviveVehicle(v);
            }
        }
    }
    
    private void reviveVehicle(Vehicle v) {
        // 1. Try to find active trip (Status 2)
        Trip trip = tripMapper.getByVehicleIdAndStatus(v.getId(), 2);
        
        // 2. Fallback: If no active trip found, try finding the latest trip
        if (trip == null) {
            Trip latestTrip = tripMapper.getByVehicleId(v.getId());
            if (latestTrip != null) {
                trip = latestTrip;
                log.warn("Vehicle {} revive: Active trip not found, using latest Trip {}", v.getId(), trip.getId());
            }
        }

        if (trip == null) {
            log.error("Vehicle {} revive failed: No Trip found. Resetting to IDLE.", v.getId());
            v.setStatus(1);
            v.setUpdateTime(LocalDateTime.now());
            vehicleMapper.update(v);
            return;
        }
        
        int segmentIndex = (v.getStatus() == 2) ? 1 : 2;
        
        // 3. Try load route from Redis
        List<Double[]> points = routeStorageService.loadRoute(trip.getId(), segmentIndex);
        
        // 4. Fallback: Re-plan route if missing using TripSegment info
        if (points == null || points.isEmpty()) {
            log.info("Vehicle {} revive: Route missing for Seg {}. Attempting re-plan using Segment info...", v.getId(), segmentIndex);
            points = replanRouteUsingSegment(trip, segmentIndex, v);
        }

        if (points != null && !points.isEmpty()) {
            vehicleRouteManager.startRoute(v.getId(), points);
            log.info("Vehicle {} revived successfully (Status={})", v.getId(), v.getStatus());
        } else {
            log.error("Vehicle {} revive failed: Could not load or plan route.", v.getId());
        }
    }

    private List<Double[]> replanRouteUsingSegment(Trip trip, int segmentIndex, Vehicle v) {
        try {
            // Retrieve specific segment
            TripSegment segment = getSegment(trip.getId(), segmentIndex);
            if (segment == null) {
                log.error("Re-plan failed: Segment {} not found for Trip {}", segmentIndex, trip.getId());
                return null;
            }

            // Use Segment coordinates which are precise
            String origin = segment.getBeginLon() + "," + segment.getBeginLat();
            String destination = segment.getEndLon() + "," + segment.getEndLat();
            
            log.info("Re-planning route for Vehicle {} Seg {}: {} -> {}", v.getId(), segmentIndex, origin, destination);

            Map<String, Object> route = tripUtils.planTrip(origin, destination, null);
            List<Double[]> polyline = (List<Double[]>) route.get("polyline");
            
            if (polyline != null && !polyline.isEmpty()) {
                // Save to Redis
                routeStorageService.saveRoute(trip.getId(), segmentIndex, polyline);
                
                // Update segment distance/duration in DB
                Double dist = (Double) route.get("distance");
                Double dur = (Double) route.get("duration");
                if (dist != null) segment.setDistance(dist);
                if (dur != null) segment.setDuration(dur);
                tripSegmentMapper.update(segment);
                
                return polyline;
            }
        } catch (Exception e) {
            log.error("Re-plan failed for vehicle {}", v.getId(), e);
        }
        return null;
    }
    
    private TripSegment getSegment(Long tripId, int sequence) {
        List<TripSegment> segments = tripSegmentMapper.getByTripId(tripId);
        if (segments == null) return null;
        return segments.stream().filter(s -> s.getSequence() == sequence).findFirst().orElse(null);
    }

    private void handleStationaryVehicles() {
        List<Vehicle> vehicles = vehicleService.getPendingVehicles(20); 
        
        double globalMultiplier = vehicleRouteManager.getGlobalSpeedMultiplier();
        long requiredStaySeconds = (long) (5.0 / globalMultiplier);
        if (requiredStaySeconds < 1) requiredStaySeconds = 1;

        for (Vehicle v : vehicles) {
            int status = v.getStatus();
            if (status == 3 || status == 5) {
                java.time.Duration duration = java.time.Duration.between(v.getUpdateTime(), LocalDateTime.now());
                if (duration.getSeconds() >= requiredStaySeconds) {
                    log.info("Vehicle {} finished waiting (Status={}), triggering update", v.getId(), status);
                    vehicleService.updateVehicle(v);
                }
            }
        }
    }

    private void processVehicle(Long vehicleId, VehicleRouteManager.RouteState state, double step) {
        state.setAccumulator(state.getAccumulator() + step);
        int advance = (int) state.getAccumulator();
        if (advance > 0) {
            state.setCurrentIndex(state.getCurrentIndex() + advance);
            state.setAccumulator(state.getAccumulator() - advance);
        }

        boolean finished = false;
        if (state.getCurrentIndex() >= state.getPoints().size() - 1) {
            state.setCurrentIndex(state.getPoints().size() - 1);
            finished = true;
        }

        Double[] currentPoint = state.getCurrentPoint();
        if (currentPoint != null) {
            Vehicle vehicle = new Vehicle();
            vehicle.setId(vehicleId);
            vehicle.setLon(currentPoint[0]);
            vehicle.setLat(currentPoint[1]);
            vehicle.setUpdateTime(LocalDateTime.now());

            // Calculate Angle
            Double[] nextPoint = state.getNextPoint();
            if (nextPoint != null && !finished) {
                double angle = calculateBearing(currentPoint[0], currentPoint[1], nextPoint[0], nextPoint[1]);
                vehicle.setAngle(angle);
            }
            
            try {
                vehicleMapper.update(vehicle);
            } catch (Exception e) {
                log.error("DB update failed for vehicle {}", vehicleId, e);
            }

            if (finished) {
                vehicleRouteManager.removeVehicle(vehicleId);
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
