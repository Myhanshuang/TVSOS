package com.tvsos.controller;

import com.tvsos.service.VehicleService;
import com.tvsos.service.impl.VehicleServiceImpl;
import dto.LocationReportDTO;
import dto.VehicleDTO;
import entity.Vehicle;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;
import result.Result;

import java.util.List;

@Slf4j
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
        log.info("获取硬件发送的地理位置信息...");
        vehicleService.updateVehicleLocation(
                report.getLicense(),
                report.getLon(),
                report.getLat(),
                report.getSpeed()
        );
        return Result.success();
    }

    /**
     * 筛选/获取车辆列表
     * @param vehicleDTO
     * @return
     */
    @GetMapping
    public Result list(VehicleDTO vehicleDTO){
        log.info("筛选/获取车辆列表");
        List<Vehicle> vehicleList = vehicleService.list(vehicleDTO);
        return Result.success(vehicleList);
    }

    /**
     * 根据id获取车辆信息
     * @param id
     * @return
     */
    @GetMapping("/{id}")
    public Result getById(@PathVariable("id") Long id){
        log.info("根据id获取车辆信息");
        Vehicle vehicle = vehicleService.getById(id);
        return Result.success(vehicle);
    }
}