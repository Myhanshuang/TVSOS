package com.tvsos.controller;

import com.tvsos.service.impl.VehicleServiceImpl;
import dto.LocationReportDTO;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/vehicles") // 所有请求的基础路径
public class VehicleController {

    private final VehicleServiceImpl vehicleServiceImpl;

    public VehicleController(VehicleServiceImpl vehicleServiceImpl) {
        this.vehicleServiceImpl = vehicleServiceImpl;
    }

    // 对应我们之前设计的接口：POST /api/vehicles/location/report
    @PostMapping("/location/report")
    public ResponseEntity<?> receiveLocationReport(@RequestBody LocationReportDTO report) {
        try {
            vehicleServiceImpl.updateVehicleLocation(
                    report.getLicense(),
                    report.getLon(),
                    report.getLat(),
                    report.getSpeed()
            );
            // 如果成功，返回 HTTP 200 OK 和一个简单的成功消息
            return ResponseEntity.ok().body("{\"message\": \"Location updated successfully\"}");
        } catch (RuntimeException e) {
            // 如果在Service层抛出异常（如找不到车辆），捕获并返回 HTTP 404 Not Found
            return ResponseEntity.status(404).body("{\"error\": \"" + e.getMessage() + "\"}");
        } catch (Exception e) {
            // 对于其他未知错误，返回 HTTP 500 Internal Server Error
            return ResponseEntity.status(500).body("{\"error\": \"An internal error occurred.\"}");
        }
    }
}