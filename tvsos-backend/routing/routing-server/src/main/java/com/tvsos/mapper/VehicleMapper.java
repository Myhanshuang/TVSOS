package com.tvsos.mapper;

import entity.Vehicle;

import org.apache.ibatis.annotations.Mapper;
import java.util.Optional;

@Mapper // 告诉Spring Boot这是一个MyBatis的Mapper接口
public interface VehicleMapper {

    /**
     * 根据车牌号查询车辆信息
     * @param license 车牌号
     * @return 车辆信息
     */
    Optional<Vehicle> findByLicense(String license);

    /**
     * 更新车辆信息
     * @param vehicle 包含最新信息的车辆对象
     * @return 受影响的行数
     */
    int update(Vehicle vehicle);
}