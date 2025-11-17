package com.tvsos.controller;

import com.tvsos.service.PoiService;
import dto.PoiQueryDTO;
import entity.Poi;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.*;
import result.Result;

import java.util.List;

@Slf4j
@RestController
@RequestMapping("/poi")
public class PoiController {
    @Autowired
    private PoiService poiService;

    /**
     * 筛选/获取poi列表
     * @param poiDTO
     * @return
     */
    @GetMapping
    public Result list(PoiQueryDTO poiDTO){
        log.info("获取/筛选所有poi");
        List<Poi> poiList = poiService.list(poiDTO);
        return Result.success(poiList);
    }

    /**
     * 根据id获取poi
     * @param id
     * @return
     */
    @GetMapping("/{id}")
    public Result getById(@PathVariable("id") Long id){
        log.info("根据id获取poi");
        Poi poi = poiService.getById(id);
        return Result.success(poi);
    }

}
