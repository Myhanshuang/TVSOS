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

    // @Scheduled(fixedDelayString = "#{T(java.util.concurrent.TimeUnit).SECONDS.toMillis(T(constant.VehicleConstant).UPDATE_SECOND)}")
    public void updatePendingVehicles(){
        // Disabled in favor of unified SimulationTask
    }

}
