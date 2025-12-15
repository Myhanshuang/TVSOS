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

            if(trip != null) {
                List<TripSegment> tripSegmentList = tripSegmentMapper.getByTripId(trip.getId());
                if(tripSegmentList != null && !tripSegmentList.isEmpty()) {
                    String origin = vehicle.getLon().toString() + "," + vehicle.getLat().toString();
                    TripSegment currSegment = null;
                    if(vehicle.getStatus() == 1 || vehicle.getStatus() == 2) {
                        currSegment = tripSegmentList.stream()
                                .filter(s -> s.getSequence() == 1)
                                .sorted(Comparator.comparingInt(TripSegment::getSequence))
                                .findFirst()
                                .orElse(null);
                    }else {
                        currSegment = tripSegmentList.stream()
                                .filter(s -> s.getSequence() == 2)
                                .sorted(Comparator.comparingInt(TripSegment::getSequence))
                                .findFirst()
                                .orElse(null);
                    }
                    String destination = currSegment.getEndLon().toString() + "," + currSegment.getEndLat().toString();
                    Map<String, Object> planMap = tripUtils.planTrip(origin, destination, null);
                    vehicleVO.setDistance((Double) planMap.get("distance"));
                    vehicleVO.setDuration((Double) planMap.get("duration"));
                    vehicleVO.setPolyline((List<Double[]>) planMap.get("polyline"));
                    Double speed = vehicleVO.getDistance() / vehicleVO.getDuration();
                    vehicle.setSpeed(speed);
                    vehicleMapper.update(vehicle);
                    vehicleVO.setSpeed(speed);
//                    vehicleVO.setSteps((JSONArray) planMap.get("steps"));
//                    vehicleVO.setRaw((JSONObject) planMap.get("raw"));
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
            vehicle.setStatus(MarkovStatusUtils.nextState(vehicle.getStatus()));
            vehicle.setUpdateTime(LocalDateTime.now());
            vehicleMapper.update(vehicle);
            return;
        }

        Integer status = vehicleMapper.getById(vehicleId).getStatus();
        // 装货逻辑
        if(status == 3){
            vehicle.setStatus(MarkovStatusUtils.nextState(status));
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
            return;
        }
        //卸货逻辑
        if (status == 5) {
            // 状态流转（卸货 -> 空闲）
            vehicle.setStatus(MarkovStatusUtils.nextState(status));  // 5 -> 1

            // 当前卸货段（status = 2）
            TripSegment currentSeg2 = segments.stream()
                    .filter(s -> s.getStatus() == 2)
                    .findFirst()
                    .orElse(null);

            if (currentSeg2 != null) {
                currentSeg2.setStatus(3);        // 卸货段完成
                tripSegmentMapper.update(currentSeg2);
            }

            vehicle.setUpdateTime(LocalDateTime.now());
            vehicleMapper.update(vehicle);
            return;
        }

        // ========== 关键逻辑：是否到达 segment 终点？ ==========
        LocalDateTime segBegin = getSegmentBeginTime(trip, currentSeg, segments);
        LocalDateTime shouldArriveTime =
                segBegin.plusSeconds((long) (currentSeg.getDuration() * 3600));

        LocalDateTime now = LocalDateTime.now();

        if (now.isBefore(shouldArriveTime)) {
            // ---- 未到时间：只更新 update_time ----
            vehicle.setUpdateTime(now);
            vehicleMapper.update(vehicle);
            return;
        }

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

        // 4) 如果是最后一个 segment 结束 → trip 结束
        if (currentSeg.getSequence() == 2) {
            trip.setStatus(3);
            trip.setEndTime(now);
            tripMapper.update(trip);

            // 所有 task 结束
            List<Long> taskIdList = tripTaskAssignMapper.getByTripId(trip.getId());
            for(Long taskId : taskIdList){
                Task task = new Task();
                task.setId(taskId);
                task.setStatus(3);
                taskMapper.update(task);
            }

            // 车辆回到空闲
            vehicle.setStatus(MarkovStatusUtils.nextState(vehicle.getStatus()));
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