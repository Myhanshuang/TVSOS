package com.tvsos.mapper;

import entity.TripTaskAssign;
import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Select;

@Mapper
public interface TripTaskAssignMapper {

    @Select("select tta.sequence from trip_task_assign tta where trip_id = #{id} order by sequence desc limit 1")
    Integer getMaxSequenceByTripId(Long id);

    @Insert("insert into trip_task_assign (trip_id, task_id, sequence) VALUE " +
            "(#{tripId}, #{taskId}, #{sequence})")
    void insert(TripTaskAssign tta);
}
