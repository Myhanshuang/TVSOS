package com.tvsos.mapper;

import entity.VehicleCategory;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Select;

import java.util.List;

@Mapper
public interface VehicleCategoryMapper {
    @Select("select * from vehicle_category where id = #{id}")
    VehicleCategory getById(Long id);

    @Select("select * from vehicle_category")
    List<VehicleCategory> findAll();
}
