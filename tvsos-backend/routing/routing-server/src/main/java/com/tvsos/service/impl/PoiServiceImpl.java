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
}
