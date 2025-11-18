package com.tvsos.mapper;

import entity.ShipmentCargo;
import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Select;

import java.util.List;

@Mapper
public interface ShipmentCargoMapper {
    @Insert("insert into shipment_cargo (shipment_id, cargo_id, quantity, weight) VALUE " +
            "(#{shipmentId}, #{cargoId}, #{quantity}, #{weight})")
    void insert(ShipmentCargo shipmentCargo);

    @Select("select * from shipment_cargo where shipment_id = #{shipmentId}")
    List<ShipmentCargo> getByShipmentId(Long shipmentId);
}
