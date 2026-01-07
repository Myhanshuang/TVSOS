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
        
        // 2. Fallback: If no active trip found, try finding the latest trip (maybe status was updated wrongly or query failed)
        if (trip == null) {
            Trip latestTrip = tripMapper.getByVehicleId(v.getId());
            if (latestTrip != null) {
                // Check if this trip is logically relevant (e.g., status is 2 but mapped poorly, or it's the one we need)
                // If vehicle is status 4, and trip is status 2 (or even 3?), we might need it.
                // Assuming safety: use latest trip if it looks recent.
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
        
        // 3. Try load route
        List<Double[]> points = routeStorageService.loadRoute(trip.getId(), segmentIndex);
        
        // 4. Fallback: Re-plan route if missing (The User's Request)
        if (points == null || points.isEmpty()) {
            log.info("Vehicle {} revive: Route missing for Seg {}. Attempting re-plan...", v.getId(), segmentIndex);
            points = replanRoute(trip, segmentIndex, v);
        }

        if (points != null && !points.isEmpty()) {
            vehicleRouteManager.startRoute(v.getId(), points);
            log.info("Vehicle {} revived successfully (Status={})", v.getId(), v.getStatus());
        } else {
            log.error("Vehicle {} revive failed: Could not load or plan route.", v.getId());
        }
    }

    private List<Double[]> replanRoute(Trip trip, int segmentIndex, Vehicle v) {
        try {
            String origin, destination;
            
            if (segmentIndex == 1) {
                // Pickup: Vehicle Start -> Task Begin (Trip End is usually Task End, Trip Begin is Vehicle Start)
                // BUT wait, Trip.BeginLon/Lat was set to Vehicle location at start.
                // Trip.EndLon/Lat is Task End.
                // We need Task Begin!
                // We can get it from TripSegment 1 End.
                TripSegment seg1 = getSegment(trip.getId(), 1);
                if (seg1 == null) return null;
                origin = trip.getBeginLon() + "," + trip.getBeginLat(); // Approx start
                destination = seg1.getEndLon() + "," + seg1.getEndLat();
            } else {
                // Delivery: Task Begin -> Task End
                // Task Begin is TripSegment 1 End (or Seg 2 Start).
                // Task End is Trip.EndLon/Lat.
                TripSegment seg2 = getSegment(trip.getId(), 2);
                if (seg2 != null) {
                    origin = seg2.getBeginLon() + "," + seg2.getBeginLat();
                    destination = seg2.getEndLon() + "," + seg2.getEndLat();
                } else {
                    // Fallback to Trip End
                    origin = v.getLon() + "," + v.getLat(); // Current pos
                    destination = trip.getEndLon() + "," + trip.getEndLat();
                }
            }

            Map<String, Object> route = tripUtils.planTrip(origin, destination, null);
            List<Double[]> polyline = (List<Double[]>) route.get("polyline");
            
            if (polyline != null && !polyline.isEmpty()) {
                routeStorageService.saveRoute(trip.getId(), segmentIndex, polyline);
                
                // Also update segment distance/duration
                TripSegment seg = getSegment(trip.getId(), segmentIndex);
                if (seg != null) {
                    Double dist = (Double) route.get("distance");
                    Double dur = (Double) route.get("duration");
                    seg.setDistance(dist);
                    seg.setDuration(dur);
                    tripSegmentMapper.update(seg);
                }
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

            // Angle calculation removed as Vehicle entity does not support it
            
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
}