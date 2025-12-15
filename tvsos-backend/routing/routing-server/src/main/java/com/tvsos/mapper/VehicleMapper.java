package com.tvsos.mapper;

import dto.VehicleQueryDTO;
import entity.Vehicle;

import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Select;
import vo.VehicleVO;

import java.util.List;

@Mapper // 告诉Spring Boot这是一个MyBatis的Mapper接口
public interface VehicleMapper {

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
    @Select("select * from vehicle where vehicle.id = #{id}")
    Vehicle getById(Long id);


    void update(Vehicle vehicle);

    @Select("select * from vehicle order by update_time limit #{vehicleBatchSize}")
    List<Vehicle> getPendingVehicles(Integer vehicleBatchSize);
}