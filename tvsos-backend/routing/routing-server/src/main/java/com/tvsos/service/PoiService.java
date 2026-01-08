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

    /**
     * 查询所有poi
     * @return
     */
    List<Poi> findAll();

    /**
     * 查询种类 为 tybe 的poi数量
     * @param tybe
     * @return
     */
    Integer countTybe(Integer tybe);

    /**
     * 统计 poi 数量
     * @return
     */
    Integer count();
}
