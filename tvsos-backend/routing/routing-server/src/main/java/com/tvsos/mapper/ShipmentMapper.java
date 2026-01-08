package com.tvsos.mapper;

import dto.ShipmentDTO;
import entity.Shipment;
import org.apache.ibatis.annotations.*;

import java.util.List;

@Mapper
public interface ShipmentMapper {

    @Insert("insert into shipment (num, begin_lon, begin_lat, end_lon, end_lat, est_begin_time, est_end_time, create_time) " +
            "VALUE (#{num}, #{beginLon}, #{beginLat}, #{endLon}, #{endLat}, #{estBeginTime}, #{estEndTime}, #{createTime})")
    @Options(useGeneratedKeys = true, keyProperty = "id")
    void insert(Shipment shipment);

    @Select("select * from shipment where status = 1 order by create_time limit #{batchSize}")
    List<Shipment> getPendingShipments(@Param("batchSize") int batchSize);

    void update(Shipment shipment);

    List<Shipment> list(ShipmentDTO shipmentDTO);
}
