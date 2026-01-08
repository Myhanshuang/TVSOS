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
import vo.VehicleVO;

import java.util.List;
import java.util.ArrayList;
import entity.Trip;

@Slf4j
@RestController
@RequestMapping("/vehicles") // 所有请求的基础路径
@Tag(name = "Vehicle")
public class VehicleController {

    @Autowired
    private VehicleService vehicleService;
    @Autowired
    private com.tvsos.service.RouteStorageService routeStorageService;
    @Autowired
    private com.tvsos.mapper.TripMapper tripMapper;

    /**
     * 筛选/获取车辆列表
     * @param vehicleQueryDTO
     * @return
     */
    @GetMapping
    @Operation(summary = "筛选/获取车辆列表")
    public Result<List<VehicleVO>> list(@ParameterObject VehicleQueryDTO vehicleQueryDTO){
        log.info("筛选/获取车辆列表");
        List<VehicleVO> vehicleVOList = vehicleService.list(vehicleQueryDTO);
        return Result.success(vehicleVOList);
    }
    
    @GetMapping("/path/{id}")
    @Operation(summary = "获取车辆当前的完整路径")
    public Result<List<Double[]>> getVehiclePath(@PathVariable Long id) {
        Vehicle vehicle = vehicleService.getById(id);
        if (vehicle == null) return Result.error("车辆不存在");
        
        // 只有行驶状态才有路径 (2:接单, 4:运货)
        if (vehicle.getStatus() != 2 && vehicle.getStatus() != 4) {
            return Result.success(new ArrayList<>());
        }

        Trip trip = tripMapper.getByVehicleIdAndStatus(id, 2); // 查找进行中的 trip
        if (trip == null) return Result.success(new ArrayList<>());

        // 判断是第几段
        // Status 2 (接单) -> Segment 1
        // Status 4 (运货) -> Segment 2
        int segmentIndex = (vehicle.getStatus() == 2) ? 1 : 2;

        List<Double[]> points = routeStorageService.loadRoute(trip.getId(), segmentIndex);
        return Result.success(points != null ? points : new ArrayList<>());
    }

    /**
     * 根据id获取车辆信息
     * @param id
     * @return
     */
    @GetMapping("/{id}")
    @Operation(summary = "根据id获取车辆信息")
    public Result<Vehicle> getById(@Parameter @PathVariable("id") Long id){
        log.info("根据id获取车辆信息");
        Vehicle vehicle = vehicleService.getById(id);
        return Result.success(vehicle);
    }
}