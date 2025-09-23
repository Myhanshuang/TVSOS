package com.tvsos.mapper;

import dto.PoiQueryDTO;
import entity.Poi;
import org.apache.ibatis.annotations.Mapper;

import java.util.List;

@Mapper
public interface PoiMapper {
    /**
     * 获取/筛选所有poi
     * @param poiQueryDTO
     * @return
     */
    List<Poi> list(PoiQueryDTO poiQueryDTO);
}
