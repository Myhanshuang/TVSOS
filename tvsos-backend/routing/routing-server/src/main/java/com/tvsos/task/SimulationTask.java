package com.tvsos.task;

import com.tvsos.manager.VehicleRouteManager;
import com.tvsos.mapper.TripMapper;
import com.tvsos.mapper.TripSegmentMapper;
import com.tvsos.mapper.VehicleMapper;
import com.tvsos.service.RouteStorageService;
import com.tvsos.service.VehicleService;
import com.tvsos.utils.MarkovStatusUtils;
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
    private TripSegmentMapper tripSegmentMapper;

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

        if (!activeRoutes.isEmpty()) {
            activeRoutes.forEach((vehicleId, state) -> {
                try {
                    // Calculate step based on pre-calculated speed (points/sec) and global multiplier
                    double step = state.getPointsPerSecond() * globalMultiplier;
                    processVehicle(vehicleId, state, step);
                } catch (Exception e) {
                    log.error("Simulation error for vehicle {}", vehicleId, e);
                }
            });
        }
    }
    
    private void handleReviveVehicles() {
        List<Vehicle> vehicles = vehicleService.getPendingVehicles(20);
        Map<Long, VehicleRouteManager.RouteState> activeRoutes = vehicleRouteManager.getAllActiveRoutes();
        
        for (Vehicle v : vehicles) {
            if ((v.getStatus() == 2 || v.getStatus() == 4) && !activeRoutes.containsKey(v.getId())) {
                reviveVehicle(v);
            }
        }
    }
    
    private void reviveVehicle(Vehicle v) {
        Trip trip = tripMapper.getByVehicleIdAndStatus(v.getId(), 2);
        
        if (trip == null) {
            Trip latestTrip = tripMapper.getByVehicleId(v.getId());
            if (latestTrip != null) {
                trip = latestTrip;
            }
        }

        if (trip == null) {
            v.setStatus(1);
            v.setUpdateTime(LocalDateTime.now());
            vehicleMapper.update(v);
            return;
        }
        
        int segmentIndex = (v.getStatus() == 2) ? 1 : 2;
        
        // Load route from storage
        List<Double[]> points = routeStorageService.loadRoute(trip.getId(), segmentIndex);
        
        if (points != null && !points.isEmpty()) {
            // Find duration to calculate speed
            TripSegment segment = getSegment(trip.getId(), segmentIndex);
            long durationSec = (segment != null && segment.getDuration() != null) 
                    ? (long)(segment.getDuration() * 3600) 
                    : points.size(); // Fallback to 1 pt/sec

            int startIndex = findClosestPointIndex(points, v.getLon(), v.getLat());
            
            vehicleRouteManager.startRoute(v.getId(), points, durationSec);
            VehicleRouteManager.RouteState state = vehicleRouteManager.getVehicleState(v.getId());
            if (state != null) {
                state.setCurrentIndex(startIndex);
                log.info("Vehicle {} revived at index {}/{}", v.getId(), startIndex, points.size());
            }
        } else {
            log.warn("Vehicle {} revive failed: Route missing for Seg {}. Waiting for repair or reset.", v.getId(), segmentIndex);
        }
    }

    private int findClosestPointIndex(List<Double[]> points, Double vehicleLon, Double vehicleLat) {
        if (points == null || points.isEmpty()) return 0;
        if (vehicleLon == null || vehicleLat == null) return 0;
        
        int bestIndex = 0;
        double minDistanceSq = Double.MAX_VALUE;
        
        for (int i = 0; i < points.size(); i++) {
            Double[] p = points.get(i);
            double dx = p[0] - vehicleLon;
            double dy = p[1] - vehicleLat;
            double distSq = dx*dx + dy*dy;
            
            if (distSq < minDistanceSq) {
                minDistanceSq = distSq;
                bestIndex = i;
            }
        }
        return bestIndex;
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
