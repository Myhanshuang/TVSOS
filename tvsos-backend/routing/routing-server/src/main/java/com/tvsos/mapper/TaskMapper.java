package com.tvsos.mapper;

import entity.Task;
import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Mapper;

@Mapper
public interface TaskMapper {
    @Insert("insert into task (shipment_id, cargo_id, quantity, begin_lon, begin_lat, end_lon, end_lat, status, weight) VALUE " +
            "(#{shipmentId}, #{cargoId}, #{quantity}, #{beginLon}, #{beginLat}, #{endLon}, #{endLat}, #{status}, #{weight})")
    void insert(Task task);
}
