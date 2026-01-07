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
            // 所有 segment 完成 (意味着卸货也完成了，现在在这里处理 status 5 -> 1 的逻辑)
            // 增加 5秒 延时，让状态 5 停留一会
            if (java.time.Duration.between(vehicle.getUpdateTime(), LocalDateTime.now()).getSeconds() < 5) {
                return;
            }
            
            // 所有 segment 完成 → trip 要完成
            trip.setStatus(3);
            trip.setEndTime(LocalDateTime.now());
            tripMapper.update(trip);

            // 该 trip 的所有 task 也要完成
            List<Long> taskIdList = tripTaskAssignMapper.getByTripId(trip.getId());
            for(Long taskId : taskIdList){
                Task task = new Task();
                task.setId(taskId);
                task.setStatus(3);
                taskMapper.update(task);
            }
            // 车辆回到空闲
            vehicle.setStatus(MarkovStatusUtils.nextState(vehicle.getStatus())); // 5 -> 1
            vehicle.setUpdateTime(LocalDateTime.now());
            vehicleMapper.update(vehicle);
            return;
        }

        Integer status = vehicleMapper.getById(vehicleId).getStatus();
        // 装货逻辑 (Status 3)
        if(status == 3){
            // 增加 5秒 延时
            if (java.time.Duration.between(vehicle.getUpdateTime(), LocalDateTime.now()).getSeconds() < 5) {
                return;
            }
            
            vehicle.setStatus(MarkovStatusUtils.nextState(status)); // 变为 4 (运货行驶)
            TripSegment nextSeg = segments.stream()
                    .filter(s -> s.getStatus() == 1)
                    .sorted(Comparator.comparingInt(TripSegment::getSequence))
                    .findFirst()
                    .orElse(null);
            if(nextSeg == null){
                throw new ServiceException("运货路线为空");
            }
            vehicle.setUpdateTime(LocalDateTime.now());
            nextSeg.setStatus(2);
            vehicleMapper.update(vehicle);
            tripSegmentMapper.update(nextSeg);

            // [New] 加载运货段(Segment 2)的路径并开始模拟
            List<Double[]> points = routeStorageService.loadRoute(trip.getId(), 2);
            if (points != null && !points.isEmpty()) {
                vehicleRouteManager.startRoute(vehicle.getId(), points);
            }
            return;
        }
        //卸货逻辑 - 注意：status 5 的进入是在运货段结束时。status 5 的离开是在 currentSeg == null 时。
        // 这里保留这个块，防止异常进入，但正常流程不会走这里，因为 status 5 时 currentSeg 应该是 null (如果 seg 2 已经置为 3)
        if (status == 5) {
            // 如果代码走到这里，说明 currentSeg 不为 null，说明数据状态可能有异常
            // 或者正在进行某种特殊处理。为了安全，保留原逻辑但指向正确的流转
            // 但按照新设计，status 5 应该由上面的 if (currentSeg == null) 处理
            return; 
        }

        // ========== 关键逻辑：是否到达 segment 终点？ ==========
        // [New] 移除基于时间的判断逻辑。
        // 现在由 SimulationTask 驱动，当调用 updateVehicle 时，意味着车辆已经由仿真引擎驱动到了终点。
        
        LocalDateTime now = LocalDateTime.now();

        // ========== 已到达 segment 终点：做状态流转 ==========

        // 1) 当前 segment 完成
        currentSeg.setStatus(3);
        tripSegmentMapper.update(currentSeg);

        // 2) 更新车辆位置到 segment 终点
        vehicle.setLat(currentSeg.getEndLat());
        vehicle.setLon(currentSeg.getEndLon());

        // 3) 使用 Markov 决定车辆状态
        int nextState = MarkovStatusUtils.nextState(vehicle.getStatus());
        vehicle.setStatus(nextState);

        // 4) 如果是最后一个 segment 结束
        if (currentSeg.getSequence() == 2) {
             // 此时 nextState 应该是 5 (卸货)
             // 我们只更新车辆状态为 5，不结束 Trip，也不跳转到 1
             // Trip 结束和 5->1 的跳转交给下一次轮询 (上面的 if (currentSeg == null))
        } else {
             // 如果是第一段结束 (接单完成 -> 装货)
             // nextState 应该是 3 (装货)
             // 此时也只是更新状态，不开启下一段，等待 Loading 延时
        }

        // 5) 更新时间
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