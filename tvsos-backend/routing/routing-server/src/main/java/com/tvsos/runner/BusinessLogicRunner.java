package com.tvsos.runner;

import com.tvsos.service.DispatchService;
import com.tvsos.service.OrderSplitterService;
import com.tvsos.service.TaskService;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.scheduling.annotation.EnableScheduling;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;
import java.util.Random;

@Component
@EnableScheduling
@Slf4j
public class BusinessLogicRunner {

    // 业务循环：每 10 秒执行一次
    private static final long BUSINESS_INTERVAL = 10000;
    private final Random random = new Random();

    @Autowired
    private TaskService taskService; // 阶段 1

    @Autowired
    private OrderSplitterService orderSplitterService; // 阶段 2

    @Autowired
    private DispatchService dispatchService; // 阶段 3 (我们刚写的)

    /**
     * 【慢速循环】：负责所有昂贵的数据库业务
     */
    @Scheduled(fixedRate = BUSINESS_INTERVAL, initialDelay = 5000) // 延迟5秒启动
    public void runBusinessLoop() {
        log.info("--- (慢速循环 10s) Tick: 业务逻辑处理 ---");

        // 1. 阶段 1：需求生成 (随机触发)
        if (random.nextDouble() < 0.5) { // 50%的几率
            taskService.createRandomTask();
        }

        // 2. 阶段 2：订单拆分
        orderSplitterService.splitUnprocessedOrders();

        // 3. 阶段 3：任务分配
        dispatchService.dispatchPendingTasks();
    }
}