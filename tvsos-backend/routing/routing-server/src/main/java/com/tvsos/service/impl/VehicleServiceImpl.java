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
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.stereotype.Service;
import vo.VehicleVO;

import java.time.LocalDateTime;
import java.util.*;

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

            // Fetch active or latest trip
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
                         // 1. Load Route (Polyline)
                         List<Double[]> polyline = routeStorageService.loadRoute(trip.getId(), targetSeq);
                         if (polyline != null) {
                             vehicleVO.setPolyline(polyline);
                         }

                         // 2. Calculate Remaining Distance/Duration
                         double totalDistance = currSegment.getDistance() != null ? currSegment.getDistance() : 0.0;
                         double totalDuration = currSegment.getDuration() != null ? currSegment.getDuration() : 0.0;
                         
                         double progress = 0.0;
                         // Only calculate progress for active moving states
                         if (vehicle.getStatus() == 2 || vehicle.getStatus() == 4) {
                             progress = vehicleRouteManager.getProgress(vehicle.getId());
                         } else if (vehicle.getStatus() == 3 || vehicle.getStatus() == 5) {
                             // At destination (Loading/Unloading) -> Progress 100% (Remaining 0)
                             // Or keep it 100% so distance is 0.
                             progress = 1.0;
                         } else if (vehicle.getStatus() == 1) {
                             // IDLE -> if trip finished, 0 remaining? 
                             // Usually IDLE means no active trip, but we might be showing history.
                             progress = 1.0; 
                         }
                         
                         double remainingRatio = 1.0 - progress;
                         // Clamp ratio
                         remainingRatio = Math.max(0.0, Math.min(1.0, remainingRatio));
                         
                         vehicleVO.setDistance(totalDistance * remainingRatio);
                         vehicleVO.setDuration(totalDuration * remainingRatio);
                         
                         // 3. Speed
                         if (totalDuration > 0) {
                             vehicleVO.setSpeed(totalDistance / totalDuration);
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
            // 删除路线 redis 缓存
            redisTemplate.delete("route:vehicleId:" + vehicleId);
            return;
        }

        List<TripSegment> segments = tripSegmentMapper.getByTripId(trip.getId());
        if (segments == null || segments.isEmpty()) {
            // 不合法数据，只更新时间
            vehicle.setUpdateTime(LocalDateTime.now());
            vehicleMapper.update(vehicle);
            // 删除路线 redis 缓存
            redisTemplate.delete("route:vehicleId:" + vehicleId);
            return;
        }

        // 找出尚未完成的 segment
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
            // 删除路线 redis 缓存
            redisTemplate.delete("route:vehicleId:" + vehicleId);
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
            // 删除路线 redis 缓存
            redisTemplate.delete("route:vehicleId:" + vehicleId);
            return;
        }
        //卸货逻辑
        if (status == 5) {
            // 状态流转（卸货 -> 空闲）
            vehicle.setStatus(MarkovStatusUtils.nextState(status));  // 5 -> 1

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

            vehicle.setUpdateTime(LocalDateTime.now());
            vehicleMapper.update(vehicle);
            // 删除路线 redis 缓存
            redisTemplate.delete("route:vehicleId:" + vehicleId);
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
        
        vehicle.setStatus(nextState);
        vehicle.setUpdateTime(now);
        vehicleMapper.update(vehicle);
    }

    /**
     * 查询所有车辆
     * @return
     */
    @Override
    public List<Vehicle> findAll() {
        List<Vehicle> vehicleList = vehicleMapper.findAll();
        return vehicleList;
    }

    /**
     * 查找该 categoryId 有多少辆车
     * @param categoryId
     * @return
     */
    @Override
    public Integer countVehicleCategory(Long categoryId) {
        Integer count = vehicleMapper.countVehicleCategory(categoryId);
        return count;
    }

    // 获取当前 segment 的实际开始时间
    private LocalDateTime getSegmentBeginTime(Trip trip, TripSegment currentSeg, List<TripSegment> allSegs) {
        // 第 1 段：使用 trip 的 beginTime 或 createTime
        if (currentSeg.getSequence() == 1) {
            LocalDateTime begin = trip.getBeginTime();
            if (begin == null) begin = trip.getCreateTime();
            return begin;
        }

        // 第 2 段：开始时间 = 第 1 段结束时间
        // 找到 seq = 1 的 segment
        TripSegment seg1 = allSegs.stream()
                .filter(s -> s.getSequence() == 1)
                .findFirst()
                .orElse(null);

        if (seg1 == null) throw new ServiceException("Segment 数据异常");

        LocalDateTime seg1Begin = trip.getBeginTime();
        if (seg1Begin == null) seg1Begin = trip.getCreateTime();

        long seg1Seconds = (long) (seg1.getDuration() * 3600);

        return seg1Begin.plusSeconds(seg1Seconds);
    }
}