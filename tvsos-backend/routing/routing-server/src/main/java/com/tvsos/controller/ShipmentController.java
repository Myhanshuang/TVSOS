package com.tvsos.controller;

import com.tvsos.service.ShipmentService;
import dto.ShipmentDTO;
import entity.Shipment;
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
@RequestMapping("/shipments")
@Tag(name = "Shipment")
public class ShipmentController {
    @Autowired
    private ShipmentService shipmentService;

    /**
     * mock用户订单
     * @param count
     * @return
     */
    @PostMapping("/mock/{count}")
    @Operation(summary = "mock用户订单")
    public Result<List<Shipment>> mockShipments(@Parameter @PathVariable("count") int count) {
        List<Shipment> shipmentList = shipmentService.mockShipments(count);
        log.info("mock 用户订单：{} 个", count);
        return Result.success(shipmentList);
    }

    /**
     * 查询 / 筛选用户订单
     * @return
     */
    @GetMapping
    @Operation(summary = "查询 / 筛选用户订单")
    public Result<List<Shipment>> list(@ParameterObject ShipmentDTO shipmentDTO) {
        log.info("查询 / 筛选用户订单: {}", shipmentDTO);
        List<Shipment> shipmentList = shipmentService.list(shipmentDTO);
        return Result.success(shipmentList);
    }

}
