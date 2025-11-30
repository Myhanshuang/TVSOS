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

    /**
     * 获取待分配的任务 前 taskBatchSize 个
     * @param taskBatchSize
     * @return
     */
    @Override
    public List<Task> getPendingTasks(Integer taskBatchSize) {
        List<Task> pendingTasks = taskMapper.getPendingTasks(taskBatchSize);
        return pendingTasks;
    }

    /**
     * 分配任务（核心实现）
     *
     * 规则：
     *  - 只挑选 vehicle.status == 1（可用）
     *  - 车辆剩余容量 >= task.weight （剩余 = vehicle_category.capacity - vehicle.cargo_size）
     *  - 车辆 type(scope) 能处理货物level（见 matchVehicleCapability 方法）
     *  - 在满足上述条件的车辆里，用 TripUtils.planTrip(vehPos -> task.begin) 获取 distance（km），选最小距离的车
     *
     */
    @Override
    @Transactional
    public boolean dispatchTask(Task task) {
        if (task == null) return false;

        // 1. 读取货物信息，获得货物级别
        Cargo cargo = cargoMapper.getById(task.getCargoId());
        if (cargo == null) {
            throw new ServiceException("货物未找到！ not found id = " + task.getCargoId());
        }
        int cargoLevel = cargo.getLevel(); // 1普通 2冷链 3危险品

        // 2. 获取所有可用车辆（status = 1）
        VehicleQueryDTO vehicleQueryDTO = new VehicleQueryDTO();
        vehicleQueryDTO.setStatus(1);
        List<Vehicle> candidates = vehicleMapper.list(vehicleQueryDTO);

        if (candidates == null || candidates.isEmpty()) {
            // 没有可用车辆，直接返回（下次定时任务再尝试）
            return false;
        }

        // 3. 在候选车辆里筛选：容量 & 能力匹配
        List<Vehicle> matched = new ArrayList<>();
//        Map<Long, Double> vehicleToRemainingCapacity = new HashMap<>();
        for (Vehicle v : candidates) {
            // get category
            if (v.getCategoryId() == null){
                throw new ServiceException("车辆没有类型！id = " + v.getId());
            }
            Long catId = v.getCategoryId();
            // vehicle_category capacity
            VehicleCategory cat = vehicleCategoryMapper.getById(catId);
            if (cat == null) {
                throw new ServiceException("车辆类型不存在！id = " + catId);
            }

            double capacity = cat.getCapacity(); // 最大承载 kg
            double cargoSize = v.getCargoSize() == null ? 0.0 : v.getCargoSize(); // 车辆当前载货量
            // 车辆剩余容量
            double remaining = capacity - cargoSize;
//            vehicleToRemainingCapacity.put(v.getId(), remaining);

            if (remaining < (task.getWeight() == null ? 0.0 : task.getWeight())) {
                // 不足以承载
                continue;
            }

            // 能力匹配（冷链 / 危险品）
            if (!matchVehicleCapability(cargoLevel, cat.getScope())) {
                continue;
            }

            matched.add(v);
        }

        if (matched.isEmpty()) {
            // 没有匹配车辆
            return false;
        }

        // 4. 在 matched 车辆中计算 vehicle -> task.begin 的真实驾车距离（用高德）
        double bestDistance = Double.POSITIVE_INFINITY;
        Vehicle bestVehicle = null;
        Map<String, Object> bestRouteToPickup = null;

        String destination = task.getBeginLon() + "," + task.getBeginLat();

        for (Vehicle v : matched) {
            String origin = v.getLon() + "," + v.getLat();

            // 使用 TripUtils.planTrip 来计算距离（高德）(origin->destination)
            Map<String, Object> route = tripUtils.planTrip(origin, destination, null);
            // route.distance 单位 km（TripUtils 实现返回的是 double distance）
            Double distanceKm = (Double) route.get("distance");
            if (distanceKm == null) continue;

            if (distanceKm < bestDistance) {
                bestDistance = distanceKm;
                bestVehicle = v;
                bestRouteToPickup = route;
            }
        }

        if (bestVehicle == null) {
            // 没有能到达的候选车（极少发生）
            return false;
        }

        // 5. 选择可用司机
        DriverQueryDTO driverQueryDTO = new DriverQueryDTO();
        driverQueryDTO.setStatus(1);
        List<Driver> driverList = driverMapper.list(driverQueryDTO);
        if(driverList.isEmpty()){
            return false;
        }
        Driver driver = driverList.get(0);

        // 6. 调用 TripService 生成 trip + trip_segment + trip_task_assign
        createTripForTask(task, bestVehicle, driver, bestRouteToPickup);

        // dispatchTask 内不做 Markov 状态转移，Markov 在每个阶段性 segment 完成后触发
        return true;
    }

    /**
     * 判断车辆类型（vehicle_category.scope）能否处理 cargo.level
     *
     * 规则（实现建议）：
     * - cargoLevel == 1 (普通)：任意车辆都可
     * - cargoLevel == 2 (冷链)：vehicle.scope == 2 或 4(特种设备) 可以
     * - cargoLevel == 3 (危化)：vehicle.scope == 3 或 4 可以
     *
     * scope 含义参照 vehicle_category 表注释：1普通 2冷链 3危险品 4特种设备
     */
    private boolean matchVehicleCapability(int cargoLevel, int vehicleScope) {
        if (cargoLevel == 1) return true;
        if (cargoLevel == 2) return vehicleScope == 2 || vehicleScope == 4;
        if (cargoLevel == 3) return vehicleScope == 3 || vehicleScope == 4;
        return false;
    }

    /**
     * 创建 trip 并为 task 生成 segment（仅两段：接单段 + 运货段）
     */
    private void createTripForTask(Task task, Vehicle vehicle, Driver driver, Map<String, Object> routeToPickupCached) {

        // 1. 创建 trip（行程）
        Trip trip = new Trip();
        trip.setVehicleId(vehicle.getId());
        trip.setStatus(1);
        trip.setCreateTime(LocalDateTime.now());
        trip.setBeginTime(LocalDateTime.now());

        // trip 起点 = 车辆当前位置
        trip.setBeginLon(vehicle.getLon());
        trip.setBeginLat(vehicle.getLat());

        // trip 终点 = 任务终点
        trip.setEndLon(task.getEndLon());
        trip.setEndLat(task.getEndLat());

        tripMapper.insert(trip);

        // 2. 绑定司机（可选）
        if (driver != null) {
            TripDriverAssign assign = new TripDriverAssign();
            assign.setTripId(trip.getId());
            assign.setDriverId(driver.getId());
            assign.setRole(1);
            tripMapper.insertTripDriverAssign(assign);

            driver.setStatus(2); // 执行中
            driverMapper.update(driver);
        }

        // 3. trip_task_assign
        TripTaskAssign tta = new TripTaskAssign();
        tta.setTripId(trip.getId());
        tta.setTaskId(task.getId());

        Integer maxSeq = tripTaskAssignMapper.getMaxSequenceByTripId(trip.getId());
        if (maxSeq == null) maxSeq = 0;
        tta.setSequence(maxSeq + 1);
        tripTaskAssignMapper.insert(tta);

        // 4. 规划两段路线
        Map<String, Object> routeToPickup = routeToPickupCached;
        if (routeToPickup == null) {
            routeToPickup = tripUtils.planTrip(
                    vehicle.getLon() + "," + vehicle.getLat(),
                    task.getBeginLon() + "," + task.getBeginLat(),
                    null
            );
        }

        Map<String, Object> routeDeliver = tripUtils.planTrip(
                task.getBeginLon() + "," + task.getBeginLat(),
                task.getEndLon() + "," + task.getEndLat(),
                null
        );

        // 5. 插入两条 segment（简单段，无 polyline）
        insertSimpleSegments(
                trip.getId(),
                routeToPickup,
                routeDeliver,
                vehicle,
                task
        );

        // 6. task 变运输中
        task.setStatus(2);
        if (task.getCreateTime() == null) task.setCreateTime(LocalDateTime.now());
        taskMapper.update(task);

        // 7. 车辆更新
//        double cur = vehicle.getCargoSize() == null ? 0.0 : vehicle.getCargoSize();
//        double add = task.getWeight() == null ? 0.0 : task.getWeight();
//        vehicle.setCargoSize(cur + add);
        vehicle.setStatus(2);
        vehicleMapper.update(vehicle);

        // 8. trip 状态改为行驶中
        trip.setStatus(2);
        tripMapper.update(trip);
    }

    /**
     * 新策略：trip_segment 只保存两段：
     * 1) 空驶段：vehicle -> task.begin
     * 2) 运货段：task.begin -> task.end
     */
    private void insertSimpleSegments(
            Long tripId,
            Map<String, Object> routeToPickup,
            Map<String, Object> routeDeliver,
            Vehicle vehicle,
            Task task
    ) {
        int seq = 1;

        // --- 第一段：空驶 A->B ---
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
        seg1.setStatus(2);
        seg1.setDuration(deadheadDuration);
        tripSegmentMapper.insert(seg1);

        // --- 第二段：运货 B->C ---
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
        seg2.setStatus(2);
        seg2.setDuration(deliverDuration);
        tripSegmentMapper.insert(seg2);
    }

    /** 高德路线的 steps 里距离相加（单位 km） */
    private double getRouteDistance(Map<String, Object> route) {
        Object dist = route.get("distance");
        if (dist == null) return 0.0;
        return Double.parseDouble(dist.toString()); // 单位：km
    }

    /**
     * 根据高德地图 api 判断预估时间 (单位：h)
     * @param route
     * @return
     */
    private double getRouteDuration(Map<String, Object> route) {
        Object dur = route.get("duration");
        if (dur == null) return 0.0;
        return Double.parseDouble(dur.toString()); // 单位：小时 h
    }
}
