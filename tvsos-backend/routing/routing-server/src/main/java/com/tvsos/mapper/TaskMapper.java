package com.tvsos.mapper;

import entity.Task;
import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Select;

import java.util.List;

@Mapper
public interface TaskMapper {
    @Insert("insert into task (shipment_id, cargo_id, quantity, begin_lon, begin_lat, end_lon, end_lat, status, weight, create_time) VALUE " +
            "(#{shipmentId}, #{cargoId}, #{quantity}, #{beginLon}, #{beginLat}, #{endLon}, #{endLat}, #{status}, #{weight}, #{createTime})")
    void insert(Task task);

    @Select("select * from task where status = 1 order by create_time limit #{taskBatchSize}")
    List<Task> getPendingTasks(Integer taskBatchSize);

    @Select("select * from task where shipment_id = #{shipmentId} and status = 1")
    List<Task> getPendingTasksByShipmentId(Long shipmentId);

    void update(Task task);
}
