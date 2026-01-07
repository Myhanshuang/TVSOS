package com.tvsos.controller;

import com.tvsos.mapper.TripMapper;
import com.tvsos.mapper.TripSegmentMapper;
import com.tvsos.service.RouteStorageService;
import com.tvsos.service.VehicleService;
import entity.Trip;
import entity.Vehicle;
import io.swagger.v3.oas.annotations.Operation;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;
import result.Result;

import java.util.ArrayList;
import java.util.List;

@RestController
public class VehicleController_Fragment {
    @Autowired
    private RouteStorageService routeStorageService;
    @Autowired
    private TripMapper tripMapper;
    @Autowired
    private TripSegmentMapper tripSegmentMapper;
    @Autowired
    private VehicleService vehicleService;

    @GetMapping("/path/{id}")
    @Operation(summary = "获取车辆当前的完整路径")
    public Result<List<Double[]>> getVehiclePath(@PathVariable Long id) {
        Vehicle vehicle = vehicleService.getById(id);
        if (vehicle == null) return Result.error("车辆不存在");

        // 只有行驶状态才有路径
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
}


