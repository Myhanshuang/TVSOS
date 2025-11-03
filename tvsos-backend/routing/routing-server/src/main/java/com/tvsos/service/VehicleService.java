package com.tvsos.service;

import dto.VehicleDTO;
import entity.Vehicle;

import java.util.List;

public interface VehicleService {
    /**
     * 更新车辆信息
     * @param license
     * @param lon
     * @param lat
     * @param speed
     */
    void updateVehicleLocation(String license, Double lon, Double lat, Double speed);

    /**
     * 筛选/获取车辆列表
     * @param vehicleDTO
     * @return
     */
    List<Vehicle> list(VehicleDTO vehicleDTO);
}
