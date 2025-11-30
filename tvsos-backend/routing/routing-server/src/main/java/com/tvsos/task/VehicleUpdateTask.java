package com.tvsos.task;

import com.tvsos.mapper.TripSegmentMapper;
import com.tvsos.mapper.VehicleMapper;
import com.tvsos.service.VehicleService;
import com.tvsos.utils.MarkovStatusUtils;
import constant.VehicleConstant;
import entity.*;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

import java.time.LocalDateTime;
import java.util.List;

@Slf4j
@Component
public class VehicleUpdateTask {
    @Autowired
    private VehicleService vehicleService;

    @Scheduled(fixedDelayString = "#{T(java.util.concurrent.TimeUnit).SECONDS.toMillis(T(constant.VehicleConstant).UPDATE_SECOND)}")
    public void updatePendingVehicles(){
        // 1. 查最旧的 20 台车辆
        List<Vehicle> vehicles = vehicleService.getPendingVehicles(VehicleConstant.VEHICLE_BATCH_SIZE);
        if (vehicles.isEmpty()) {
            log.info("无车辆存在！");
            return;
        }


        for (Vehicle v : vehicles) {
            log.info("更新车辆状态: 车辆id-{}", v.getId());
            vehicleService.updateVehicle(v);
        }

        log.info("本次车辆状态更新完成，总处理车辆数: {}", vehicles.size());

    }

}
