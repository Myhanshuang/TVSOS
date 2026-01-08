package com.tvsos.controller;

import com.tvsos.service.ReportService;
import com.tvsos.service.ShipmentService;
import com.tvsos.service.VehicleCategoryService;
import com.tvsos.service.VehicleService;
import dto.VehicleQueryDTO;
import entity.VehicleCategory;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;
import result.Result;
import vo.VehicleCategoryVO;
import vo.VehicleVO;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

@RestController
@Slf4j
@RequestMapping("/report")
@Tag(name = "Report")
public class ReportController {
    @Autowired
    private ReportService reportService;

    /**
     * 饼图 车辆类型 - 数量
     * @return
     */
    @GetMapping("/vehicleCategory")
    @Operation(summary = "饼图 车辆类型 - 数量")
    public Result<List<VehicleCategoryVO>> reportVehicleCategory() {
        log.info("饼图 车辆类型 - 数量");
        List<VehicleCategoryVO> list = reportService.reportVehicleCategory();
        return Result.success(list);
    }

    /**
     * 柱状图 poi类型 - 数量
     * @return
     */
    @GetMapping("/poiTybe")
    @Operation(summary = "柱状图 poi类型 - 数量")
    public Result<List<List>> reportPoiTybe() {
        log.info("柱状图 poi类型 - 数量");
        List<List> list = reportService.reportPoiTybe();
        return Result.success(list);
    }

    /**
     * 统计 车上的货物总量
     * @return
     */
    @GetMapping("/cargoSize")
    @Operation(summary = "统计 车上的货物总量 /kg")
    public Result<Double> reportCargoSize() {
        Double sum = reportService.reportCargoSize();
        log.info("统计目前车上的货物总量: {} kg", sum);
        return Result.success(sum);
    }

    /**
     * 统计 poi 数量
     * @return
     */
    @GetMapping("/poiSum")
    @Operation(summary = "统计 poi 数量")
    public Result<Integer> reportPoiSum(){
        Integer sum = reportService.reportPoiSum();
        log.info("统计 poi 数量: {}", sum);
        return Result.success(sum);
    }

    /**
     * 统计车辆数量
     * @return
     */
    @GetMapping("/vehicleSum")
    @Operation(summary = "统计车辆数量")
    public Result<Integer> reportVehicleSum(){
        Integer sum = reportService.reportVehicleSum();
        log.info("统计车辆数量: {}", sum);
        return Result.success(sum);
    }

    /**
     * 统计司机数量
     * @return
     */
    @GetMapping("/driverSum")
    @Operation(summary = "统计司机数量")
    public Result<Integer> reportDriverSum(){
        Integer sum = reportService.reportDriverSum();
        log.info("统计司机数量: {}", sum);
        return Result.success(sum);
    }

}
