package com.tvsos.mapper;

import dto.DriverQueryDTO;
import entity.Driver;
import org.apache.ibatis.annotations.Mapper;

import java.util.List;

@Mapper
public interface DriverMapper {
    List<Driver> list(DriverQueryDTO driverQueryDTO);

    void update(Driver driver);
}
