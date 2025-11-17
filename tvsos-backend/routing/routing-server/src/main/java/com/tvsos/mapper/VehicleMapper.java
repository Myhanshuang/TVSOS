package com.tvsos.mapper;

import dto.VehicleDTO;
import entity.Vehicle;

import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Select;

import java.util.List;
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

    /**
     * 筛选/获取车辆列表
     * @param vehicleDTO
     * @return
     */
    List<Vehicle> list(VehicleDTO vehicleDTO);

    /**
     * 根据id获取车辆信息
     * @param id
     * @return
     */
    @Select("select * from vehicle where vehicle.id = #{id}")
    Vehicle getById(Long id);
}