package com.tvsos.controller;


import dto.ModbusVehicleDTO;
import dto.ModbusFrontendDTO;
import com.tvsos.service.VehicleStateService;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.Collection;

/**
 * 车辆位置 API 控制器 (内存模式)
 */
@RestController
@RequestMapping("/api/vehicles")
@RequiredArgsConstructor
@CrossOrigin
public class ModbusController {

    private final VehicleStateService vehicleStateService;

    /**
     * 端点一：接收车辆位置上报并更新内存状态
     * POST /api/vehicles/location/report
     */
    @PostMapping("/location/report")
    public ResponseEntity<String> reportLocation(@RequestBody ModbusVehicleDTO reportDTO) {
        vehicleStateService.updateVehicleState(reportDTO);
        return ResponseEntity.ok("State for " + reportDTO.getLicense() + " updated in memory.");
    }

    /**
     * 端点二：供前端轮询，从内存中直接获取所有车辆的最新状态
     * GET /api/vehicles/latest-locations
     */
    @GetMapping("/latest-locations")
    public ResponseEntity<Collection<ModbusFrontendDTO>> getLatestLocations() {
        Collection<ModbusFrontendDTO> latestStates = vehicleStateService.getAllLatestStates();
        return ResponseEntity.ok(latestStates);
    }
}