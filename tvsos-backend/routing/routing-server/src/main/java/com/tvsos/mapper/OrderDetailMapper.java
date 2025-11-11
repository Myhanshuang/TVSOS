package com.tvsos.mapper;

import entity.OrderDetail;
import org.apache.ibatis.annotations.Mapper;
import java.util.List;

@Mapper
public interface OrderDetailMapper {
    void insert(OrderDetail orderDetail);
    List<OrderDetail> findByTransportOrderId(Long transportOrderId);
    /**
     * [新] 根据状态查询子任务列表
     */
    List<OrderDetail> findByStatus(Integer status);

    /**
     * [新] 更新子任务 (例如更新状态)
     */
    int update(OrderDetail orderDetail);
}