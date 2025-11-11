package com.tvsos.service.impl;

import com.tvsos.mapper.ContentCargoMapper;
import com.tvsos.mapper.OrderDetailMapper;
import com.tvsos.mapper.TransportOrderMapper;
import com.tvsos.service.OrderSplitterService;
import constant.StatusConstant;
import entity.ContentCargo;
import entity.OrderDetail;
import entity.TransportOrder;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.List;

@Service
@Slf4j
public class OrderSplitterServiceImpl implements OrderSplitterService {

    private final TransportOrderMapper transportOrderMapper;
    private final OrderDetailMapper orderDetailMapper;
    private final ContentCargoMapper contentCargoMapper;

    // 车辆载重 (吨)，从大到小
    // 按照某老师最初的想法搞得
    private static final double[] VEHICLE_CAPACITIES = {18.0, 9.0, 4.5};

    public OrderSplitterServiceImpl(TransportOrderMapper transportOrderMapper,
                                    OrderDetailMapper orderDetailMapper,
                                    ContentCargoMapper contentCargoMapper) {
        this.transportOrderMapper = transportOrderMapper;
        this.orderDetailMapper = orderDetailMapper;
        this.contentCargoMapper = contentCargoMapper;
    }

    /**
     * [修改] 非事务性 - 查找并分发待处理的订单
     */
    @Override
    public void splitUnprocessedOrders() {
        // 1. 查找所有“待处理”的原始总订单
        List<TransportOrder> unprocessedOrders = transportOrderMapper.findByStatus(StatusConstant.ORDER_UNPROCESSED);

        for (TransportOrder order : unprocessedOrders) {
            // 2. [修改] 查找该订单的 "所有" 货物内容
            List<ContentCargo> contents = contentCargoMapper.findByTransportOrderId(order.getId());

            if (contents.isEmpty()) {
                log.warn("【阶段2】拆分跳过: TransportOrder Id={} 缺少 ContentCargo。将标记为已处理。", order.getId());
                // 为防止重复查询空订单，将其标记为已处理
                order.setStatus(StatusConstant.ORDER_PROCESSED);
                transportOrderMapper.update(order);
                continue;
            }

            try {
                // 3. [修改] 调用新的事务方法，处理该订单的 "所有" 货物内容
                processOrderSplitting(order, contents);

            } catch (Exception e) {
                log.error("【阶段2】订单拆分失败 (事务回滚): TransportOrder Id={}", order.getId(), e);
                // 事务回滚，订单状态保持 UNPROCESSED，等待下次循环重试
            }
        }
    }

    /**
     * [新] 事务性 - 完整拆分一个总订单及其所有内容
     * * @param order 总订单
     * @param contents 该订单包含的所有货物
     */
//    @Transactional
    public void processOrderSplitting(TransportOrder order, List<ContentCargo> contents) {
        log.info("【阶段2】订单拆分: 正在处理 TransportOrder Id={}, 包含 {} 个货物项",
                order.getId(), contents.size());

        // [!] 核心修改：遍历 "所有" 货物内容
        for (ContentCargo content : contents) {

            log.info(" -> 拆分内容: CargoId={}, 总重量={}t", content.getCargoId(), content.getQuantity());

            double remainingWeight = content.getQuantity();

            // 4. 贪心算法拆分 (18t, 9t, 4.5t)
            for (double capacity : VEHICLE_CAPACITIES) {
                while (remainingWeight >= capacity) {
                    // 创建一个 18t / 9t / 4.5t 的子任务
                    createSubTask(order, content.getCargoId(), (int) capacity);
                    remainingWeight -= capacity;
                }
            }

            // 5. 处理剩余的“零头”
            if (remainingWeight > 0.1) { // 避免0.0x的浮点问题
                createSubTask(order, content.getCargoId(), (int) Math.ceil(remainingWeight));
            }
        } // 结束对 contents 的循环

        // 6. [!] 将原始总订单标记为“已处理”
        // 只有在 "所有" 货物都成功拆分（事务未回滚）时，才更新总订单状态
        order.setStatus(StatusConstant.ORDER_PROCESSED);
        transportOrderMapper.update(order);

        log.info("【阶段2】订单拆分完成: TransportOrder Id={}", order.getId());
    }

    /**
     * 辅助方法：创建一条“可调度”的子任务 (OrderDetail)
     */
    private void createSubTask(TransportOrder order, Long cargoId, int weight) {
        OrderDetail subTask = new OrderDetail();
        subTask.setTransportOrderId(order.getId());
        subTask.setCargoId(cargoId);
        subTask.setQuantity(weight); // 存入 18, 9, 4.5 或 零头
        subTask.setStatus(StatusConstant.TASK_READY_FOR_DISPATCH); // 1: 待调度

        orderDetailMapper.insert(subTask); //
        log.info("    -> 创建子任务 (OrderDetail): Weight={}, Status={}", weight, StatusConstant.TASK_READY_FOR_DISPATCH);
    }
}