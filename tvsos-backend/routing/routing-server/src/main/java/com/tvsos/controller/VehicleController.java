package com.tvsos.controller;

import com.tvsos.service.VehicleService;
import dto.VehicleQueryDTO;
import entity.Vehicle;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.Parameter;
import io.swagger.v3.oas.annotations.tags.Tag;
import lombok.extern.slf4j.Slf4j;
import org.springdoc.core.annotations.ParameterObject;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.*;
import result.Result;

import java.util.List;

@Slf4j
@RestController
@RequestMapping("/vehicles") // 所有请求的基础路径
@Tag(name = "Vehicle")
public class VehicleController {

    @Autowired
    private VehicleService vehicleService;

    /**
     * 筛选/获取车辆列表
     * @param vehicleQueryDTO
     * @return
     */
    @GetMapping
    @Operation(summary = "筛选/获取车辆列表")
    public Result list(@ParameterObject VehicleQueryDTO vehicleQueryDTO){
        log.info("筛选/获取车辆列表");
        List<Vehicle> vehicleList = vehicleService.list(vehicleQueryDTO);
        return Result.success(vehicleList);
    }

    /**
     * 根据id获取车辆信息
     * @param id
     * @return
     */
    @GetMapping("/{id}")
    @Operation(summary = "根据id获取车辆信息")
    public Result getById(@Parameter @PathVariable("id") Long id){
        log.info("根据id获取车辆信息");
        Vehicle vehicle = vehicleService.getById(id);
        return Result.success(vehicle);
    }
}