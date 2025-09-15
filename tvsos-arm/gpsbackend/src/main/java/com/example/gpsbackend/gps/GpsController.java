package com.example.gpsbackend.gps;

import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;
import org.springframework.beans.factory.annotation.Autowired;

@RestController
public class GpsController {

    private final GpsService gpsService;

    @Autowired
    public GpsController(GpsService gpsService) {
        this.gpsService = gpsService;
    }

    @GetMapping("/")
    public String home() {
        System.out.println("收到来自Python的HTTP连接测试请求。");

        // 设置Python连接状态为已连接
        gpsService.setPythonConnected(true);

        return "Spring Boot Backend is running!";
    }

    @PostMapping("/api/gps_data")
    public ResponseEntity<String> receiveGpsData(@RequestBody GpsData gpsData) {
        //数据接收示例（此处只将其打印在控制台上）
        System.out.println("\n**************************************************");
        System.out.println(" 收到来自Python的GPS数据:");
        System.out.println(" 纬度 (Latitude):" + gpsData.getLatitude());
        System.out.println(" 经度 (Longitude):" + gpsData.getLongitude());
        System.out.println(" 速度 (Speed):" + gpsData.getSpeed());
        System.out.println(" 设备ID (Device ID):" + gpsData.getDevice_id());
        System.out.println(" 时间戳 (Timestamp):" + String.format("%.6f", gpsData.getTimestamp()));
        System.out.println(" >**************************************************\n");

        // 确保数据接收时保持连接状态
        gpsService.setPythonConnected(true);

        return new ResponseEntity<>("GPS数据接收成功！", HttpStatus.OK);
    }
}