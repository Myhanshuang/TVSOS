package com.tvsos.mapper;

import entity.TransportOrder;
import org.apache.ibatis.annotations.Mapper;
import java.util.List;

@Mapper
public interface TransportOrderMapper {
    /**
     * 插入一条新的运输任务，并返回自增ID
     */
    void insert(TransportOrder transportOrder);

    /**
     * 根据状态查询任务列表
     */
    List<TransportOrder> findByStatus(Integer status);

    //
    
    /**
     * 更新任务信息（例如：状态）
     */
    int update(TransportOrder transportOrder);

    TransportOrder findById(Long transportId);
}