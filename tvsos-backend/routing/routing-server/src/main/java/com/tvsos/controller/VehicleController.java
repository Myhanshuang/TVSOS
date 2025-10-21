package com.tvsos.controller;

import com.tvsos.service.VehicleService;
import com.tvsos.service.impl.VehicleServiceImpl;
import dto.LocationReportDTO;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;
import result.Result;

@RestController
@RequestMapping("/vehicles") // 所有请求的基础路径
public class VehicleController {

    @Autowired
    private VehicleService vehicleService;

    /**
     * 获取硬件发送的地理位置信息
     * @param report
     * @return
     */
    @PostMapping("/location/report")
    public Result receiveLocationReport(@RequestBody LocationReportDTO report){
        vehicleService.updateVehicleLocation(
                report.getLicense(),
                report.getLon(),
                report.getLat(),
                report.getSpeed()
        );
        return Result.success();
    }
}