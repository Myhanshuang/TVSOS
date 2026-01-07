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

@Service
@Slf4j
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

    /**
     * 筛选/获取车辆列表
     * @param vehicleQueryDTO
     * @return
     */
    @Override
    public List<VehicleVO> list(VehicleQueryDTO vehicleQueryDTO) {
        List<Vehicle> vehicleList = vehicleMapper.list(vehicleQueryDTO);
        List<VehicleVO> vehicleVOList = new ArrayList<>();
        for(Vehicle vehicle : vehicleList) {
            VehicleVO vehicleVO = new VehicleVO();
            BeanUtils.copyProperties(vehicle, vehicleVO);

            Trip trip =  tripMapper.getByVehicleId(vehicle.getId());

            // 移除实时路径规划逻辑，直接返回数据库中的状态
            // distance 和 duration 应该在任务创建时写入，这里不再动态计算
            // 前端只接收当前位置，不需要完整 polyline
            
            // 为了保持兼容性，如果数据库里有 distance/duration，可以设置进去
            // 这里假设 Vehicle 实体或者 TripSegment 里有相关信息
            if(trip != null) {
                List<TripSegment> tripSegmentList = tripSegmentMapper.getByTripId(trip.getId());
                if(tripSegmentList != null && !tripSegmentList.isEmpty()) {
                     TripSegment currSegment = null;
                     if(vehicle.getStatus() == 1 || vehicle.getStatus() == 2) { // 状态1应该是空闲，这里可能是之前的逻辑有问题，先保留原判断结构
                         currSegment = tripSegmentList.stream()
                                 .filter(s -> s.getSequence() == 1)
                                 .sorted(Comparator.comparingInt(TripSegment::getSequence))
                                 .findFirst()
                                 .orElse(null);
                     } else {
                         currSegment = tripSegmentList.stream()
                                 .filter(s -> s.getSequence() == 2)
                                 .sorted(Comparator.comparingInt(TripSegment::getSequence))
                                 .findFirst()
                                 .orElse(null);
                     }
                     if (currSegment != null) {
                         // 1. 获取基础总数据
                         double totalDistance = currSegment.getDistance() != null ? currSegment.getDistance() : 0.0;
                         double totalDuration = currSegment.getDuration() != null ? currSegment.getDuration() : 0.0;
                         
                         // 2. 获取实时仿真进度
                         double progress = 0.0;
                         // 只有在行驶状态下才去查进度
                         if (vehicle.getStatus() == 2 || vehicle.getStatus() == 4) {
                             progress = vehicleRouteManager.getProgress(vehicle.getId());
                         } else if (vehicle.getStatus() == 3 || vehicle.getStatus() == 5) {
                             // 装卸货状态，距离视为 0 或保持到达状态
                             progress = 1.0; 
                         }

                         // 3. 计算剩余数据
                         double remainingRatio = 1.0 - progress;
                         vehicleVO.setDistance(totalDistance * remainingRatio);
                         vehicleVO.setDuration(totalDuration * remainingRatio);
                         
                         // 4. 设置速度 (km/h)
                         // 如果总时间有效，用 总距离/总时间 计算平均速度
                         // 或者直接给一个固定值 (例如 60km/h)，因为仿真速度是倍率控制的
                         if (totalDuration > 0) {
                             vehicleVO.setSpeed(totalDistance / totalDuration);
                         } else {
                             vehicleVO.setSpeed(0.0);
                         }
                     }
                }
            }
            vehicleVOList.add(vehicleVO);
        }
        return vehicleVOList;
    }

    /**
     * 根据id获取车辆信息
     * @param id
     * @return
     */
    @Override
    public Vehicle getById(Long id) {
        Vehicle vehicle = vehicleMapper.getById(id);
        return vehicle;
    }

    /**
     * 获取要更新的车辆列表
     * @param vehicleBatchSize
     * @return
     */
    @Override
    public List<Vehicle> getPendingVehicles(Integer vehicleBatchSize) {
        List<Vehicle> vehicleList = vehicleMapper.getPendingVehicles(vehicleBatchSize);
        return vehicleList;
    }

    /**
     * 更新车辆状态
     * @param vehicle
     * @return
     */
    @Override
    public void updateVehicle(Vehicle vehicle) {

        Long vehicleId = vehicle.getId();
        // 1. 查该车当前的 trip（status = 2 的）
        Trip trip = tripMapper.getByVehicleIdAndStatus(vehicleId, 2);
        if (trip == null || vehicle.getStatus() == 1) {
            // 没任务，车辆空闲，只更新更新时间
            vehicle.setUpdateTime(LocalDateTime.now());
            vehicleMapper.update(vehicle);
            return;
        }

        // 2. 查该 trip 的两个 segment，按顺序排好
        List<TripSegment> segments = tripSegmentMapper.getByTripId(trip.getId());
        if (segments == null || segments.isEmpty()) {
            // 不合法数据，只更新时间
            vehicle.setUpdateTime(LocalDateTime.now());
            vehicleMapper.update(vehicle);
            return;
        }

        // 找出尚未完成的 segment
        TripSegment currentSeg = segments.stream()
                .filter(s -> s.getStatus() == 2)
                .sorted(Comparator.comparingInt(TripSegment::getSequence))
                .findFirst()
                .orElse(null);

        if (currentSeg == null) {
            // 所有 segment 完成 (意味着卸货也完成了，现在处理 5 -> 1)
            // Trip 结束
            trip.setStatus(3);
            trip.setEndTime(LocalDateTime.now());
            tripMapper.update(trip);

            // Task 结束
            List<Long> taskIdList = tripTaskAssignMapper.getByTripId(trip.getId());
            for(Long taskId : taskIdList){
                Task task = new Task();
                task.setId(taskId);
                task.setStatus(3);
                taskMapper.update(task);
            }
            // 车辆回到空闲 (5 -> 1)
            vehicle.setStatus(MarkovStatusUtils.nextState(vehicle.getStatus())); 
            vehicle.setUpdateTime(LocalDateTime.now());
            vehicleMapper.update(vehicle);
            return;
        }

        Integer status = vehicleMapper.getById(vehicleId).getStatus();
        
        // 装货逻辑 (Status 3 -> 4)
        if(status == 3){
            // 状态流转 3 -> 4
            vehicle.setStatus(MarkovStatusUtils.nextState(status)); 
            
            // 开启下一段 Segment
            TripSegment nextSeg = segments.stream()
                    .filter(s -> s.getStatus() == 1)
                    .sorted(Comparator.comparingInt(TripSegment::getSequence))
                    .findFirst()
                    .orElse(null);
            
            if(nextSeg == null){
                throw new ServiceException("运货路线为空");
            }
            
            vehicle.setUpdateTime(LocalDateTime.now());
            nextSeg.setStatus(2); // 运货段开始
            vehicleMapper.update(vehicle);
            tripSegmentMapper.update(nextSeg);

            // [Revert & Fix] 从存储加载预先规划好的运货路径 (Segment 2)
            List<Double[]> points = routeStorageService.loadRoute(trip.getId(), 2);
            if (points != null && !points.isEmpty()) {
                // 1. 启动仿真
                vehicleRouteManager.startRoute(vehicle.getId(), points);
                
                // 2. [Critical] 强制将车辆位置同步到路径起点
                // 防止因之前的误差导致车辆“脱轨”
                Double[] startPoint = points.get(0);
                vehicle.setLon(startPoint[0]);
                vehicle.setLat(startPoint[1]);
                vehicleMapper.update(vehicle);
                
                log.info("车辆 {} 开始运货，强制归位到路径起点: [{}, {}]", vehicle.getId(), startPoint[0], startPoint[1]);
            } else {
                log.error("严重错误：车辆 {} 无法加载运货路径 (TripId={}, Segment=2)", vehicle.getId(), trip.getId());
            }
            return;
        }
        
        // 卸货逻辑 (Status 5 -> 1) 
        // 实际上 status 5 是由 SimulationTask 在 Segment 2 结束时设置的
        // 这里如果是 status 5 且 currentSeg 不为空，说明逻辑有点错位，
        // 但通常 SimulationTask 会在此时调用 updateVehicle 触发 5 -> 1 (如果 currentSeg 全部完成了)
        // 上面的 if (currentSeg == null) 会处理大部分 5->1 的情况
        if (status == 5) {
             // 保护性逻辑，如果还有未完成的 segment 但状态是 5，可能需要强制修正
             return; 
        }

        // ========== 移动结束逻辑 (由 SimulationTask 触发) ==========
        // 当 SimulationTask 发现车辆走到 Segment 终点时调用此方法
        
        LocalDateTime now = LocalDateTime.now();

        // 1) 当前 segment 完成
        currentSeg.setStatus(3);
        tripSegmentMapper.update(currentSeg);

        // 2) 更新车辆位置到 segment 终点
        vehicle.setLat(currentSeg.getEndLat());
        vehicle.setLon(currentSeg.getEndLon());

        // 3) 状态流转
        // 如果是第一段 (接单) 结束 (2 -> 3)
        // 如果是第二段 (运货) 结束 (4 -> 5)
        int nextState = MarkovStatusUtils.nextState(vehicle.getStatus());
        vehicle.setStatus(nextState);

        // 4) 更新时间
        vehicle.setUpdateTime(now);
        vehicleMapper.update(vehicle);
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