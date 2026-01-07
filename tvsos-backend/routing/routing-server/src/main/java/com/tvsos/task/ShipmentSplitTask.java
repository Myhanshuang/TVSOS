package com.tvsos.task;

import com.tvsos.service.ShipmentService;
import constant.ShipmentConstant;
import entity.Shipment;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

import java.util.List;

/**
 * 用户订单拆分定时任务
 */
@Slf4j
@Component
public class ShipmentSplitTask {

    @Autowired
    private ShipmentService shipmentService;

    /**
     * 每隔 SCHEDULE_MINUTE 分钟执行一次
     * 自动拆分订单 -> 任务
     */
    // @Scheduled(fixedDelayString = "#{T(java.util.concurrent.TimeUnit).MINUTES.toMillis(T(constant.ShipmentConstant).SCHEDULE_MINUTE)}")
    public void splitPendingShipments() {
        log.info("开始执行订单拆分定时任务...(间隔：{} min，最大拆分订单：{} 个)", ShipmentConstant.SCHEDULE_MINUTE, ShipmentConstant.SHIPMENT_BATCH_SIZE);

        try {
            // 获取前 batchSize 个待拆分订单
            List<Shipment> shipments = shipmentService.getPendingShipments(ShipmentConstant.SHIPMENT_BATCH_SIZE);

            if (shipments == null || shipments.isEmpty()) {
                log.info("没有待拆分订单");
                return;
            }

            for (Shipment shipment : shipments) {
                log.info("拆分订单: {}", shipment.getNum());
                shipmentService.splitShipment(shipment, ShipmentConstant.TASK_WEIGHT_LIMIT);
            }

            log.info("本次订单拆分完成，总处理订单数: {}", shipments.size());

        } catch (Exception e) {
            log.error("订单拆分定时任务执行异常", e);
        }
    }
}
