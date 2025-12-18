package com.tvsos.controller;

import com.tvsos.service.PoiService;
import dto.PoiQueryDTO;
import entity.Poi;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.Parameter;
import io.swagger.v3.oas.annotations.tags.Tag;
import lombok.extern.slf4j.Slf4j;
import org.springdoc.core.annotations.ParameterObject;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.*;
import result.Result;

import java.util.List;

@Slf4j
@RestController
@RequestMapping("/poi")
@Tag(name = "Poi")
public class PoiController {
    @Autowired
    private PoiService poiService;

    /**
     * 筛选/获取poi列表
     * @param poiDTO
     * @return
     */
    @GetMapping
    @Operation(summary = "筛选/获取poi列表")
    public Result<List<Poi>> list(@ParameterObject PoiQueryDTO poiDTO){
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
    @Operation(summary = "根据id获取poi")
    public Result<Poi> getById(@Parameter @PathVariable("id") Long id){
        log.info("根据id获取poi");
        Poi poi = poiService.getById(id);
        return Result.success(poi);
    }

}
