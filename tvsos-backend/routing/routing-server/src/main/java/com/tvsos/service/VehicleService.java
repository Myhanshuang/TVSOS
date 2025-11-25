package com.tvsos.service;

import dto.VehicleQueryDTO;
import entity.Vehicle;

import java.util.List;

public interface VehicleService {
//    /**
//     * 更新车辆信息
//     * @param license
//     * @param lon
//     * @param lat
//     * @param speed
//     */
//    void updateVehicleLocation(String license, Double lon, Double lat, Double speed);

    /**
     * 筛选/获取车辆列表
     * @param vehicleQueryDTO
     * @return
     */
    List<Vehicle> list(VehicleQueryDTO vehicleQueryDTO);

    /**
     * 根据id获取车辆信息
     * @param id
     * @return
     */
    Vehicle getById(Long id);


    /**
     * 获取要更新的车辆列表
     * @param vehicleBatchSize
     * @return
     */
    List<Vehicle> getPendingVehicles(Integer vehicleBatchSize);

    /**
     * 更新车辆状态
     * @param vehicle
     * @return
     */
    void updateVehicle(Vehicle vehicle);
}
