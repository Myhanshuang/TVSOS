package com.tvsos.service;

import dto.PoiQueryDTO;
import entity.Poi;

import java.util.List;

public interface PoiService {
    /**
     * 获取/筛选所有poi
     * @param poiQueryDTO
     * @return
     */
    List<Poi> list(PoiQueryDTO poiQueryDTO);

    /**
     * 根据id获取poi
     * @param id
     * @return
     */
    Poi getById(Long id);
}
