package com.tvsos.service;

import entity.VehicleCategory;

import java.util.List;

public interface VehicleCategoryService {

    /**
     * 查询所有车辆类型
     * @return
     */
    List<VehicleCategory> findAll();
}
