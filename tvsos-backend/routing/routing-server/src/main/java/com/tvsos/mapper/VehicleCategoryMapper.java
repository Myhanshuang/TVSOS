package com.tvsos.mapper;

import entity.VehicleCategory;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Select;

@Mapper
public interface VehicleCategoryMapper {
    @Select("select * from vehicle_category where id = #{id}")
    VehicleCategory getById(Long id);
}
