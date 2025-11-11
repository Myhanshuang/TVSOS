package com.tvsos.service.impl;

import com.tvsos.mapper.*;
import com.tvsos.service.DispatchService;
import com.tvsos.utils.HttpUtils;
import constant.StatusConstant;
import dto.VehicleDTO;
import entity.*;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.LocalDateTime;
import java.util.List;
import java.util.Optional;

@Service
@Slf4j
public class DispatchServiceImpl implements DispatchService {

    private final OrderDetailMapper orderDetailMapper;
    private final TransportOrderMapper transportOrderMapper;
    private final VehicleMapper vehicleMapper;
    private final PoiMapper poiMapper;
    private final CategoryMapper categoryMapper;
    private final AssignMapper assignMapper;

    public DispatchServiceImpl(OrderDetailMapper orderDetailMapper, TransportOrderMapper transportOrderMapper,
                               VehicleMapper vehicleMapper, PoiMapper poiMapper,
                               CategoryMapper categoryMapper, AssignMapper assignMapper) {
        this.orderDetailMapper = orderDetailMapper;
        this.transportOrderMapper = transportOrderMapper;
        this.vehicleMapper = vehicleMapper;
        this.poiMapper = poiMapper;
        this.categoryMapper = categoryMapper;
        this.assignMapper = assignMapper;
    }

    @Override
    @Transactional
    public void dispatchPendingTasks() {
        // 1. 找出所有“待调度”的子任务 (Status = 1)
        List<OrderDetail> pendingJobs = orderDetailMapper.findByStatus(StatusConstant.TASK_READY_FOR_DISPATCH);
        if (pendingJobs.isEmpty()) return;

        // 2. 找出所有“空闲”的车辆 (Status = 6)
        VehicleDTO filter = new VehicleDTO();
        filter.setStatus(StatusConstant.FREE);
        List<Vehicle> availableVehicles = vehicleMapper.list(filter);
        if (availableVehicles.isEmpty()) return;

        log.info("【阶段3】任务调度: 发现 {} 个待调度任务 和 {} 辆空闲车", pendingJobs.size(), availableVehicles.size());

        for (OrderDetail job : pendingJobs) {
            TaskRequirements reqs = calculateTaskRequirements(job);
            if (reqs == null) continue;

            // 4. 寻找最佳车辆 (匹配重量 + 距离最近)
            Vehicle bestVehicle = findBestVehicle(reqs, availableVehicles);
            //注意，这个方法是后续的修改点

            if (bestVehicle != null) {
                log.info(" -> 匹配成功: 任务 ({}t) 分配给车辆 Id={}", reqs.totalWeight, bestVehicle.getId());


                assignTaskToVehicle(job, bestVehicle, reqs.startPoi, reqs.endPoi);

                availableVehicles.remove(bestVehicle);
//                log.info("task finished");
            }
        }
    }
// 下面后面大概率要修改，现阶段先 ai 生成一个版本
    /**
     * 辅助方法：执行分配 (采用您“查找并更新”的逻辑)
     */
    private void assignTaskToVehicle(OrderDetail job, Vehicle vehicle, Poi startPoi, Poi endPoi) {
        // 1. 路线规划
        // Map<String, String> params = Map.of("origin", ..., "destination", ...);
        // String routeJson = HttpUtils.doGet("YOUR_GIS_API_URL", params); // [!] 静态调用


        // 查找车辆-司机的持久化绑定
        Optional<Assign> assignOpt = assignMapper.findByVehicleId(vehicle.getId());
        if (assignOpt.isEmpty()) {
            log.error("【阶段3】分配失败: 车辆 Id={} 缺少在 assign 表中的绑定记录", vehicle.getId());
            return; // 跳过此任务
        }

        Assign assign = assignOpt.get();

        assign.setTransportOrderId(job.getTransportOrderId());
        // assign.setOrderDetailId(job.getId()); // (如果 Assign 表有此字段)
        assignMapper.update(assign);

        // 4. 更新 子任务(Job) 状态 -> "已分配" (2)
        job.setStatus(StatusConstant.TASK_ASSIGNED);
        orderDetailMapper.update(job);

        // 5. 更新 车辆 状态 -> "接单行驶" (1)
        vehicle.setStatus(StatusConstant.DRIVING_TO_PICKUP);
        vehicle.setUpdateTime(LocalDateTime.now());
        vehicleMapper.update(vehicle);
    }

    // (辅助方法: calculateTaskRequirements, findBestVehicle, calculateDistance 保持不变)

    /**
     * 辅助方法：获取子任务的详细需求
     */
    private TaskRequirements calculateTaskRequirements(OrderDetail job) {
        double totalWeight = job.getQuantity();
        TransportOrder order = transportOrderMapper.findById(job.getTransportOrderId());
        if(order == null) return null;
        Poi startPoi = poiMapper.findById(order.getBeginPoiId());
        Poi endPoi = poiMapper.findById(order.getEndPoiId());
        if (startPoi == null || endPoi == null) return null;
        return new TaskRequirements(totalWeight, startPoi, endPoi);
    }

    /**
     * 辅助方法：寻找最佳匹配车辆
     */
    private Vehicle findBestVehicle(TaskRequirements reqs, List<Vehicle> availableVehicles) {
        Vehicle bestMatch = null;
        double minDistance = Double.MAX_VALUE;
        for (Vehicle vehicle : availableVehicles) {
            Optional<Category> vehicleCategoryOpt = categoryMapper.findById(vehicle.getId());
            if (vehicleCategoryOpt.isEmpty()) continue;
            Category vehicleCategory = vehicleCategoryOpt.get();
            if (vehicleCategory.getWeight() >= reqs.totalWeight) {
                double distance = calculateDistance(vehicle.getLat(), vehicle.getLon(),
                        reqs.startPoi.getLat(), reqs.startPoi.getLon());
                if (distance < minDistance) {
                    minDistance = distance;
                    bestMatch = vehicle;
                }
            }
        }
        return bestMatch;
    }

    /**
     * 辅助内部类：用于封装任务需求
     */
    private static class TaskRequirements {
        double totalWeight;
        Poi startPoi, endPoi;
        TaskRequirements(double w, Poi sp, Poi ep) {
            this.totalWeight = w; this.startPoi = sp; this.endPoi = ep;
        }
    }

    /**
     * 辅助方法：计算球面距离 (Haversine)
     */
    private double calculateDistance(double lat1, double lon1, double lat2, double lon2) {
        double R = 6371; // km
        double latDistance = Math.toRadians(lat2 - lat1);
        double lonDistance = Math.toRadians(lon2 - lon1);
        double a = Math.sin(latDistance / 2) * Math.sin(latDistance / 2)
                + Math.cos(Math.toRadians(lat1)) * Math.cos(Math.toRadians(lat2))
                * Math.sin(lonDistance / 2) * Math.sin(lonDistance / 2);
        double c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
        return R * c;
    }
}