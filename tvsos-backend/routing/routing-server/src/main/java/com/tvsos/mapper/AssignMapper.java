package com.tvsos.mapper;

import entity.Assign;
import org.apache.ibatis.annotations.Mapper;
import java.util.Optional; // [!] 引入

@Mapper
public interface AssignMapper {
    void insert(Assign assign);

    /**
     * [新] 根据车辆ID查找当前的分配状态
     */
    Optional<Assign> findByVehicleId(Long vehicleId);

    /**
     * [新] 更新分配表 (用于绑定任务)
     */
    int update(Assign assign);
}