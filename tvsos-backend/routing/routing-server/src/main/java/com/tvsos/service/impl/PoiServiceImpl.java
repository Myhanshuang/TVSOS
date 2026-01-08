package com.tvsos.service.impl;

import com.tvsos.mapper.PoiMapper;
import com.tvsos.service.PoiService;
import dto.PoiQueryDTO;
import entity.Poi;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

import java.util.List;

@Service
public class PoiServiceImpl implements PoiService {
    @Autowired
    private PoiMapper poiMapper;
    /**
     * 获取/筛选所有poi
     * @param poiQueryDTO
     * @return
     */
    @Override
    public List<Poi> list(PoiQueryDTO poiQueryDTO) {
        List<Poi> poiList = poiMapper.list(poiQueryDTO);
        return poiList;
    }

    /**
     * 根据id获取poi
     * @param id
     * @return
     */
    @Override
    public Poi getById(Long id) {
        Poi poi = poiMapper.getById(id);
        return poi;
    }

    /**
     * 查询所有poi
     * @return
     */
    @Override
    public List<Poi> findAll() {
        List<Poi> poiList = poiMapper.findAll();
        return poiList;
    }

    /**
     *  查询种类为 tybe 的poi数量
     * @param tybe
     * @return
     */
    @Override
    public Integer countTybe(Integer tybe) {
        return poiMapper.countTybe(tybe);
    }

    /**
     * 统计 poi 数量
     * @return
     */
    @Override
    public Integer count() {
        Integer sum = poiMapper.count();
        if(sum == null){
            sum = 0;
        }
        return sum;
    }
}
