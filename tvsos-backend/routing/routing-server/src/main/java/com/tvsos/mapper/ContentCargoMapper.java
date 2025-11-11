package com.tvsos.mapper;

import entity.ContentCargo;
import org.apache.ibatis.annotations.Mapper;
import java.util.List;

@Mapper
public interface ContentCargoMapper {
    /**
     * 插入订单内容
     */
    void insert(ContentCargo contentCargo);

    /**
     * 根据总订单ID查询内容
     */
    List<ContentCargo> findByTransportOrderId(Long transportOrderId);
}