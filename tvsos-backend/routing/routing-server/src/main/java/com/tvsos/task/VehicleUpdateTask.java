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
            return; // 日志太吵，去掉了
        }

        for (Vehicle v : vehicles) {
            // [Fix] 关键修改：如果是行驶状态 (2=接单行驶, 4=运货行驶)，跳过！
            // 它们的流转由 SimulationTask (仿真引擎) 负责，这里只负责静止状态的时间流转 (如 3=装货, 5=卸货)
            if (v.getStatus() == 2 || v.getStatus() == 4) {
                continue;
            }
            
            log.info("更新车辆状态(静止/作业中): 车辆id-{}", v.getId());
            vehicleService.updateVehicle(v);
        }
    }

}
