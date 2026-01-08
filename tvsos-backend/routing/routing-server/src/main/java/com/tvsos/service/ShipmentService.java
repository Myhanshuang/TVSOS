package com.tvsos.service;

import dto.ShipmentDTO;
import entity.Shipment;

import java.util.List;

public interface ShipmentService {
    /**
     * mock用户订单
     * @param count
     * @return
     */
    List<Shipment> mockShipments(int count);

    /**
     * 获取待拆分的订单，按创建时间升序，数量 batchSize
     */
    List<Shipment> getPendingShipments(int batchSize);

    /**
     * 拆分订单成最小颗粒度任务
     * @param shipment 待拆分订单
     * @param weightLimit 单个任务货物重量上限
     */
    void splitShipment(Shipment shipment, double weightLimit);

    /**
     * 查询 / 筛选用户订单
     * @return
     */
    List<Shipment> list(ShipmentDTO shipmentDTO);
}
