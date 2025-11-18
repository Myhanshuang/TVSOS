package com.tvsos.controller;

import com.tvsos.service.ShipmentService;
import entity.Shipment;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.Parameter;
import io.swagger.v3.oas.annotations.tags.Tag;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;
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
    public Result mockShipments(@Parameter @PathVariable("count") int count) {
        List<Shipment> shipmentList = shipmentService.mockShipments(count);
        return Result.success(shipmentList);
    }

}
