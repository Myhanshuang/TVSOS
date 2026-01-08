package com.tvsos.service.impl;

import com.tvsos.mapper.VehicleCategoryMapper;
import com.tvsos.service.VehicleCategoryService;
import entity.VehicleCategory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

import java.util.List;

@Service
public class VehicleCategoryServiceImpl implements VehicleCategoryService {
    @Autowired
    private VehicleCategoryMapper vehicleCategoryMapper;

    /**
     * 查询所有车辆类型
     * @return
     */
    @Override
    public List<VehicleCategory> findAll() {
        List<VehicleCategory> vehicleCategoryList = vehicleCategoryMapper.findAll();
        return vehicleCategoryList;
    }
}
