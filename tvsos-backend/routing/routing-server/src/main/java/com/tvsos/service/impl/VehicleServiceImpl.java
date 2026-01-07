package com.tvsos.service.impl;

import com.alibaba.fastjson2.JSONArray;
import com.alibaba.fastjson2.JSONObject;
import com.tvsos.mapper.*;
import com.tvsos.service.VehicleService;
import com.tvsos.utils.MarkovStatusUtils;
import com.tvsos.utils.TripUtils;
import dto.VehicleQueryDTO;
import entity.Task;
import entity.Trip;
import entity.TripSegment;
import entity.Vehicle;
import exception.ServiceException;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import vo.VehicleVO;

import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;
import java.util.Map;

@Slf4j
@Service
public class VehicleServiceImpl implements VehicleService {

    @Autowired
    private VehicleMapper vehicleMapper;
    @Autowired
    private TripMapper tripMapper;
    @Autowired
    private TaskMapper taskMapper;
    @Autowired
    private TripSegmentMapper tripSegmentMapper;
    @Autowired
    private TripTaskAssignMapper tripTaskAssignMapper;
    @Autowired
    private TripUtils tripUtils;
    @Autowired
    private com.tvsos.service.RouteStorageService routeStorageService;
    @Autowired
    private com.tvsos.manager.VehicleRouteManager vehicleRouteManager;

    @Override
    public List<VehicleVO> list(VehicleQueryDTO vehicleQueryDTO) {
        List<Vehicle> vehicleList = vehicleMapper.list(vehicleQueryDTO);
        List<VehicleVO> vehicleVOList = new ArrayList<>();
        
        for(Vehicle vehicle : vehicleList) {
            VehicleVO vehicleVO = new VehicleVO();
            BeanUtils.copyProperties(vehicle, vehicleVO);

            // Use getByVehicleIdAndStatus to find active trip, or fallback to latest
            // But for display, usually active is preferred.
            // If IDLE, we might show nothing or last trip.
            // Using getByVehicleId (latest) as per previous fix
            Trip trip = tripMapper.getByVehicleId(vehicle.getId());
            
            if(trip != null) {
                List<TripSegment> tripSegmentList = tripSegmentMapper.getByTripId(trip.getId());
                if(tripSegmentList != null && !tripSegmentList.isEmpty()) {
                     // Status 1(IDLE), 2(PICKUP) -> Segment 1
                     // Status 3(LOADING), 4(DELIVERY), 5(UNLOADING) -> Segment 2
                     int targetSeq = (vehicle.getStatus() <= 2) ? 1 : 2;
                     
                     TripSegment currSegment = tripSegmentList.stream()
                             .filter(s -> s.getSequence() == targetSeq)
                             .findFirst()
                             .orElse(null);

                     if (currSegment != null) {
                         vehicleVO.setDistance(currSegment.getDistance());
                         vehicleVO.setDuration(currSegment.getDuration());
                         
                         List<Double[]> polyline = routeStorageService.loadRoute(trip.getId(), targetSeq);
                         if (polyline != null) {
                             vehicleVO.setPolyline(polyline);
                         }
                         
                         if (vehicleVO.getDuration() != null && vehicleVO.getDuration() > 0) {
                             vehicleVO.setSpeed(vehicleVO.getDistance() / vehicleVO.getDuration());
                         }
                     }
                }
            }
            vehicleVOList.add(vehicleVO);
        }
        return vehicleVOList;
    }

    @Override
    public Vehicle getById(Long id) {
        return vehicleMapper.getById(id);
    }

    @Override
    public List<Vehicle> getPendingVehicles(Integer vehicleBatchSize) {
        return vehicleMapper.getPendingVehicles(vehicleBatchSize);
    }

    @Override
    public void updateVehicle(Vehicle vehicle) {
        Long vehicleId = vehicle.getId();
        Trip trip = tripMapper.getByVehicleIdAndStatus(vehicleId, 2);
        
        if (trip == null) {
            if (vehicle.getStatus() != 1) {
                vehicle.setStatus(MarkovStatusUtils.nextState(vehicle.getStatus()));
            }
            vehicle.setUpdateTime(LocalDateTime.now());
            vehicleMapper.update(vehicle);
            return;
        }

        List<TripSegment> segments = tripSegmentMapper.getByTripId(trip.getId());
        TripSegment currentSeg = segments.stream()
                .filter(s -> s.getStatus() == 2)
                .findFirst()
                .orElse(null);

        if (currentSeg == null) {
            // All segments done
            trip.setStatus(3);
            trip.setEndTime(LocalDateTime.now());
            tripMapper.update(trip);

            List<Long> taskIdList = tripTaskAssignMapper.getByTripId(trip.getId());
            for(Long taskId : taskIdList){
                Task task = new Task();
                task.setId(taskId);
                task.setStatus(3);
                taskMapper.update(task);
            }
            
            vehicle.setStatus(MarkovStatusUtils.nextState(vehicle.getStatus())); 
            vehicle.setUpdateTime(LocalDateTime.now());
            vehicleMapper.update(vehicle);
            return;
        }

        int status = vehicle.getStatus();
        
        // 1. Loading Finished (3 -> 4)
        if(status == 3){
            TripSegment nextSeg = segments.stream()
                    .filter(s -> s.getStatus() == 1)
                    .sorted(Comparator.comparingInt(TripSegment::getSequence))
                    .findFirst()
                    .orElse(null);
            
            if(nextSeg == null){
                log.error("No next segment found for vehicle {} at status 3", vehicle.getId());
                return;
            }
            
            int nextState = MarkovStatusUtils.nextState(status); // 3 -> 4
            vehicle.setStatus(nextState);
            vehicle.setUpdateTime(LocalDateTime.now());
            
            nextSeg.setStatus(2);
            tripSegmentMapper.update(nextSeg);
            vehicleMapper.update(vehicle);

            // Load Route for Simulation (Segment 2)
            List<Double[]> points = routeStorageService.loadRoute(trip.getId(), 2);
            if (points != null && !points.isEmpty()) {
                Double[] start = points.get(0);
                vehicle.setLon(start[0]);
                vehicle.setLat(start[1]);
                vehicleMapper.update(vehicle);
                
                // Pass duration for speed calc
                long durationSec = (nextSeg.getDuration() != null) ? (long)(nextSeg.getDuration() * 3600) : points.size();
                vehicleRouteManager.startRoute(vehicle.getId(), points, durationSec);
                
                log.info("Vehicle {} started delivery (Seg 2)", vehicle.getId());
            } else {
                log.error("Vehicle {} missing route for Seg 2!", vehicle.getId());
            }
            return;
        }
        
        // 2. Unloading Finished (5 -> 1)
        if (status == 5) {
             int nextState = MarkovStatusUtils.nextState(status);
             vehicle.setStatus(nextState);
             vehicle.setUpdateTime(LocalDateTime.now());
             vehicleMapper.update(vehicle);
             return; 
        }

        // 3. Arrival at Segment End
        LocalDateTime now = LocalDateTime.now();

        currentSeg.setStatus(3);
        tripSegmentMapper.update(currentSeg);

        vehicle.setLat(currentSeg.getEndLat());
        vehicle.setLon(currentSeg.getEndLon());

        int nextState = MarkovStatusUtils.nextState(vehicle.getStatus());
        
        // Removed Just-In-Time Planning block. 
        // Route for Seg 2 is assumed to be planned at dispatch.

        vehicle.setStatus(nextState);
        vehicle.setUpdateTime(now);
        vehicleMapper.update(vehicle);
    }
}
