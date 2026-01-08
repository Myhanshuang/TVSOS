package com.tvsos.controller;

import com.tvsos.manager.VehicleRouteManager;
import result.Result;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.*;

@Slf4j
@RestController
@RequestMapping("/simulation")
@Tag(name = "仿真控制接口")
public class SimulationController {

    @Autowired
    private VehicleRouteManager vehicleRouteManager;

    @PostMapping("/speed")
    @Operation(summary = "设置全局仿真倍速")
    public Result setSpeed(@RequestParam Double multiplier) {
        if (multiplier == null || multiplier <= 0) {
            return Result.error("倍速必须大于0");
        }
        vehicleRouteManager.setGlobalSpeedMultiplier(multiplier);
        return Result.success();
    }
    
    @GetMapping("/speed")
    @Operation(summary = "获取当前仿真倍速")
    public Result<Double> getSpeed() {
        return Result.success(vehicleRouteManager.getGlobalSpeedMultiplier());
    }
}
