package com.tvsos.service.impl;

import com.tvsos.mapper.VehicleMapper;
import com.tvsos.service.VehicleService;
import dto.VehicleQueryDTO;
import entity.Vehicle;
import org.springframework.stereotype.Service;

import java.util.List;

@Service
public class VehicleServiceImpl implements VehicleService {

    private final VehicleMapper vehicleMapper;

    public VehicleServiceImpl(VehicleMapper vehicleMapper) {
        this.vehicleMapper = vehicleMapper;
    }

    /**
     * 筛选/获取车辆列表
     * @param vehicleQueryDTO
     * @return
     */
    @Override
    public List<Vehicle> list(VehicleQueryDTO vehicleQueryDTO) {
        List<Vehicle> vehicleList = vehicleMapper.list(vehicleQueryDTO);
        return vehicleList;
    }

    /**
     * 根据id获取车辆信息
     * @param id
     * @return
     */
    @Override
    public Vehicle getById(Long id) {
        Vehicle vehicle = vehicleMapper.getById(id);
        return vehicle;
    }
}