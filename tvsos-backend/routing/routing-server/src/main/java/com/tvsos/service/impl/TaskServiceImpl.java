package com.tvsos.service.impl;

import com.alibaba.fastjson2.JSONObject;
import com.tvsos.mapper.*;
import com.tvsos.service.TaskService;
import com.tvsos.utils.TripUtils;
import dto.DriverQueryDTO;
import dto.VehicleQueryDTO;
import entity.*;
import exception.ServiceException;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import com.alibaba.fastjson2.JSONArray;

import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;

@Service
public class TaskServiceImpl implements TaskService {

    @Autowired
    private TaskMapper taskMapper;
    @Autowired
    private CargoMapper cargoMapper;
    @Autowired
    private VehicleMapper vehicleMapper;
    @Autowired
    private DriverMapper driverMapper;
    @Autowired
    private VehicleCategoryMapper vehicleCategoryMapper;
    @Autowired
    private TripMapper tripMapper;
    @Autowired
    private TripTaskAssignMapper tripTaskAssignMapper;
    @Autowired
    private TripSegmentMapper tripSegmentMapper;
    @Autowired
    private TripUtils tripUtils;
    @Autowired
    private com.tvsos.service.RouteStorageService routeStorageService;
    @Autowired
    private com.tvsos.manager.VehicleRouteManager vehicleRouteManager;

    @Override
    public List<Task> getPendingTasks(Integer taskBatchSize) {
        List<Task> pendingTasks = taskMapper.getPendingTasks(taskBatchSize);
        return pendingTasks;
    }

    @Override
    @Transactional
    public boolean dispatchTask(Task task) {
        if (task == null) return false;

        Cargo cargo = cargoMapper.getById(task.getCargoId());
        if (cargo == null) {
            throw new ServiceException("货物未找到！ not found id = " + task.getCargoId());
        }
        int cargoLevel = cargo.getLevel();

        VehicleQueryDTO vehicleQueryDTO = new VehicleQueryDTO();
        vehicleQueryDTO.setStatus(1);
        List<Vehicle> candidates = vehicleMapper.list(vehicleQueryDTO);

        if (candidates == null || candidates.isEmpty()) {
            return false;
        }

        List<Vehicle> matched = new ArrayList<>();
        for (Vehicle v : candidates) {
            if (v.getCategoryId() == null){
                throw new ServiceException("车辆没有类型！id = " + v.getId());
            }
            Long catId = v.getCategoryId();
            VehicleCategory cat = vehicleCategoryMapper.getById(catId);
            if (cat == null) {
                throw new ServiceException("车辆类型不存在！id = " + catId);
            }

            double capacity = cat.getCapacity();
            double cargoSize = v.getCargoSize() == null ? 0.0 : v.getCargoSize();
            double remaining = capacity - cargoSize;

            if (remaining < (task.getWeight() == null ? 0.0 : task.getWeight())) {
                continue;
            }

            if (!matchVehicleCapability(cargoLevel, cat.getScope())) {
                continue;
            }

            matched.add(v);
        }

        if (matched.isEmpty()) {
            return false;
        }

        double bestDistance = Double.POSITIVE_INFINITY;
        Vehicle bestVehicle = null;
        Map<String, Object> bestRouteToPickup = null;

        String destination = task.getBeginLon() + "," + task.getBeginLat();

        for (Vehicle v : matched) {
            String origin = v.getLon() + "," + v.getLat();
            Map<String, Object> route = tripUtils.planTrip(origin, destination, null);
            Double distanceKm = (Double) route.get("distance");
            if (distanceKm == null) continue;

            if (distanceKm < bestDistance) {
                bestDistance = distanceKm;
                bestVehicle = v;
                bestRouteToPickup = route;
            }
        }

        if (bestVehicle == null) {
            return false;
        }

        DriverQueryDTO driverQueryDTO = new DriverQueryDTO();
        driverQueryDTO.setStatus(1);
        List<Driver> driverList = driverMapper.list(driverQueryDTO);
        if(driverList.isEmpty()){
            return false;
        }
        Driver driver = driverList.get(0);

        createTripForTask(task, bestVehicle, driver, bestRouteToPickup);
        return true;
    }

    private boolean matchVehicleCapability(int cargoLevel, int vehicleScope) {
        if (cargoLevel == 1) return true;
        if (cargoLevel == 2) return vehicleScope == 2 || vehicleScope == 4;
        if (cargoLevel == 3) return vehicleScope == 3 || vehicleScope == 4;
        return false;
    }

    private void createTripForTask(Task task, Vehicle vehicle, Driver driver, Map<String, Object> routeToPickupCached) {
        Trip trip = new Trip();
        trip.setVehicleId(vehicle.getId());
        trip.setStatus(1);
        trip.setCreateTime(LocalDateTime.now());
        trip.setBeginTime(LocalDateTime.now());
        trip.setBeginLon(vehicle.getLon());
        trip.setBeginLat(vehicle.getLat());
        trip.setEndLon(task.getEndLon());
        trip.setEndLat(task.getEndLat());
        tripMapper.insert(trip);

        if (driver != null) {
            TripDriverAssign assign = new TripDriverAssign();
            assign.setTripId(trip.getId());
            assign.setDriverId(driver.getId());
            assign.setRole(1);
            tripMapper.insertTripDriverAssign(assign);
            driver.setStatus(2);
            driverMapper.update(driver);
        }

        TripTaskAssign tta = new TripTaskAssign();
        tta.setTripId(trip.getId());
        tta.setTaskId(task.getId());
        Integer maxSeq = tripTaskAssignMapper.getMaxSequenceByTripId(trip.getId());
        if (maxSeq == null) maxSeq = 0;
        tta.setSequence(maxSeq + 1);
        tripTaskAssignMapper.insert(tta);

        Map<String, Object> routeToPickup = routeToPickupCached;
        if (routeToPickup == null) {
            routeToPickup = tripUtils.planTrip(
                    vehicle.getLon() + "," + vehicle.getLat(),
                    task.getBeginLon() + "," + task.getBeginLat(),
                    null
            );
        }

        // [Modified] Plan delivery route IMMEDIATELY
        Map<String, Object> routeDeliver = tripUtils.planTrip(
                task.getBeginLon() + "," + task.getBeginLat(),
                task.getEndLon() + "," + task.getEndLat(),
                null
        );

        insertSimpleSegments(
                trip.getId(),
                routeToPickup,
                routeDeliver,
                vehicle,
                task
        );

        task.setStatus(2);
        if (task.getCreateTime() == null) task.setCreateTime(LocalDateTime.now());
        taskMapper.update(task);

        vehicle.setStatus(2);
        vehicleMapper.update(vehicle);

        trip.setStatus(2);
        tripMapper.update(trip);
    }

    private void insertSimpleSegments(
            Long tripId,
            Map<String, Object> routeToPickup,
            Map<String, Object> routeDeliver,
            Vehicle vehicle,
            Task task
    ) {
        int seq = 1;

        // --- Segment 1: Pickup ---
        double deadheadDist = getRouteDistance(routeToPickup);
        double deadheadDuration = getRouteDuration(routeToPickup);
        TripSegment seg1 = new TripSegment();
        seg1.setTripId(tripId);
        seg1.setSequence(seq++);
        seg1.setBeginLon(vehicle.getLon());
        seg1.setBeginLat(vehicle.getLat());
        seg1.setEndLon(task.getBeginLon());
        seg1.setEndLat(task.getBeginLat());
        seg1.setDistance(deadheadDist);
        seg1.setStatus(2); // In progress
        seg1.setDuration(deadheadDuration);
        tripSegmentMapper.insert(seg1);

        // Start Simulation for Segment 1
        if (routeToPickup != null) {
            List<Double[]> polyline = (List<Double[]>) routeToPickup.get("polyline");
            if (polyline != null && !polyline.isEmpty()) {
                routeStorageService.saveRoute(tripId, 1, polyline);
                // Pass duration for speed calculation
                long durationSec = (long)(deadheadDuration * 3600);
                vehicleRouteManager.startRoute(vehicle.getId(), polyline, durationSec);
            }
        }

        // --- Segment 2: Delivery ---
        double deliverDist = getRouteDistance(routeDeliver);
        double deliverDuration = getRouteDuration(routeDeliver);
        
        TripSegment seg2 = new TripSegment();
        seg2.setTripId(tripId);
        seg2.setSequence(seq);
        seg2.setBeginLon(task.getBeginLon());
        seg2.setBeginLat(task.getBeginLat());
        seg2.setEndLon(task.getEndLon());
        seg2.setEndLat(task.getEndLat());
        seg2.setDistance(deliverDist);
        seg2.setStatus(1); // Pending
        seg2.setDuration(deliverDuration);
        tripSegmentMapper.insert(seg2);
        
        // [Modified] Save Segment 2 route IMMEDIATELY
        if (routeDeliver != null) {
            List<Double[]> polyline2 = (List<Double[]>) routeDeliver.get("polyline");
            if (polyline2 != null && !polyline2.isEmpty()) {
                routeStorageService.saveRoute(tripId, 2, polyline2);
            }
        }
    }

    private double getRouteDistance(Map<String, Object> route) {
        if (route == null) return 0.0;
        Object dist = route.get("distance");
        if (dist == null) return 0.0;
        return Double.parseDouble(dist.toString());
    }

    private double getRouteDuration(Map<String, Object> route) {
        if (route == null) return 0.0;
        Object dur = route.get("duration");
        if (dur == null) return 0.0;
        return Double.parseDouble(dur.toString());
    }
}