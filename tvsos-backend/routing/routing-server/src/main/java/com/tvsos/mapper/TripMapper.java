package com.tvsos.mapper;

import entity.Trip;
import entity.TripDriverAssign;
import org.apache.ibatis.annotations.*;

@Mapper
public interface TripMapper {

    @Insert("insert into trip (vehicle_id, begin_time, end_time, create_time, begin_lat, end_lat, begin_lon, end_lon)" +
            "values (#{vehicleId}, #{beginTime}, #{endTime}, #{createTime}, #{beginLat}, #{endLat}, #{beginLon}, #{endLon})")
    @Options(useGeneratedKeys = true, keyProperty = "id", keyColumn = "id")
    void insert(Trip trip);

    @Insert("insert into trip_driver_assign (trip_id, driver_id, role) VALUE " +
            "(#{tripId}, #{driverId}, #{role})")
    void insertTripDriverAssign(TripDriverAssign assign);

    void update(Trip trip);

    @Select("select * from trip where vehicle_id = #{vehicleId} and status = #{status}")
    Trip getByVehicleIdAndStatus(@Param("vehicleId") Long vehicleId,@Param("status") Integer status);
}
