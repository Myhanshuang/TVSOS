package com.tvsos.service;

public interface OrderSplitterService {
    /**
     * 查找所有“待处理”的总订单 (TransportOrder)，
     * 并将它们拆分为适合车辆装载的“可调度”子任务 (OrderDetail)。
     * 注意，现在订单拆分的实现是非常简单的，未来可能需要根据实际业务需求进行优化。
     */
    void splitUnprocessedOrders();
}