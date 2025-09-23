package com.tvsos.service.impl;

import com.tvsos.mapper.VehicleMapper;
import com.tvsos.service.VehicleService;
import entity.Vehicle;
import org.springframework.stereotype.Service;
import java.time.LocalDateTime;

@Service
public class VehicleServiceImpl implements VehicleService {

    private final VehicleMapper vehicleMapper;

    public VehicleServiceImpl(VehicleMapper vehicleMapper) {
        this.vehicleMapper = vehicleMapper;
    }

    public void updateVehicleLocation(String license, Double lon, Double lat, Double speed) {
        // 1. 调用Mapper方法查找车辆
        Vehicle vehicle = vehicleMapper.findByLicense(license)
                .orElseThrow(() -> new RuntimeException("未找到车牌号为 " + license + " 的车辆"));

        // 2. 更新车辆对象的属性
        vehicle.setLon(lon);
        vehicle.setLat(lat);
        vehicle.setSpeed(speed);

        if (vehicle.getStatus() != null && vehicle.getStatus() == 1) {
            vehicle.setStatus(2);
        }

        // 3. 手动设置更新时间！
        vehicle.setUpdateTime(LocalDateTime.now());

        // 4. 调用Mapper方法，将更新后的整个对象传入
        vehicleMapper.update(vehicle);
    }
}