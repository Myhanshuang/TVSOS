package com.tvsos.controller;

import com.tvsos.service.PoiService;
import dto.PoiQueryDTO;
import entity.Poi;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;
import result.Result;

import java.util.List;

@Slf4j
@RestController
@RequestMapping("/poi")
public class PoiController {
    @Autowired
    private PoiService poiService;

    /**
     * 获取/筛选所有poi
     * @return
     */
    @GetMapping
    public Result list(PoiQueryDTO poiDTO){
        log.info("获取/筛选所有poi");
        List<Poi> poiList = poiService.list(poiDTO);
        return Result.success(poiList);
    }
}
