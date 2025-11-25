package com.tvsos.mapper;

import entity.TripSegment;
import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Mapper;

@Mapper
public interface TripSegmentMapper {

    @Insert("insert into trip_segment (trip_id, begin_lon, begin_lat, end_lon, end_lat, distance, sequence, status, duration) VALUE " +
            "(#{tripId}, #{beginLon}, #{beginLat}, #{endLon}, #{endLat}, #{distance}, #{sequence}, #{status}, #{duration})")
    void insert(TripSegment seg);
}
