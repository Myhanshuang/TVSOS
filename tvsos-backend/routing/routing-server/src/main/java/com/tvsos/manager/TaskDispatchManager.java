package com.tvsos.manager;

import com.tvsos.service.TaskService;
import entity.Task;
import jakarta.annotation.PostConstruct;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;

import java.util.concurrent.BlockingQueue;
import java.util.concurrent.LinkedBlockingQueue;

/**
 * 任务分配调度管理器
 * 职责：缓存待分配任务，并进行 API 调用速率限制 (Rate Limiting)
 */
@Slf4j
@Component
public class TaskDispatchManager {

    @Autowired
    private TaskService taskService;

    // 内存阻塞队列：作为缓冲区
    private final BlockingQueue<Task> taskQueue = new LinkedBlockingQueue<>();

    // 最小调用间隔 (毫秒)
    // 3 QPS => 333ms/次。设置为 400ms 以留有余地，避免临界值被拒
    private static final long RATE_LIMIT_INTERVAL = 400;

    /**
     * 提交任务 (生产者接口)
     * 该方法是非阻塞的，调用后立即返回
     */
    public void submitTask(Task task) {
        if (task == null) return;
        boolean success = taskQueue.offer(task);
        if (success) {
            log.info("任务已入列等待分配: ID={} 当前队列长度: {}", task.getId(), taskQueue.size());
        } else {
            log.error("任务队列已满，丢弃任务: ID={}", task.getId());
        }
    }

    /**
     * 初始化消费者线程
     */
    @PostConstruct
    public void initConsumer() {
        Thread consumerThread = new Thread(this::consumeLoop, "RateLimited-Dispatcher");
        consumerThread.setDaemon(true); // 设置为守护线程
        consumerThread.start();
    }

    /**
     * 消费循环 (消费者逻辑)
     * 串行处理 + 强制休眠
     */
    private void consumeLoop() {
        log.info("任务分配限流消费者已启动 (间隔 {}ms)", RATE_LIMIT_INTERVAL);
        while (true) {
            try {
                // 1. 获取任务 (如果队列为空，此处会阻塞等待)
                Task task = taskQueue.take();

                long startTime = System.currentTimeMillis();

                // 2. 执行分配逻辑 (包含高德 API 调用)
                try {
                    taskService.dispatchTask(task);
                } catch (Exception e) {
                    log.error("任务分配异常: ID={}", task.getId(), e);
                }

                // 3. 速率控制 (Rate Limiting)
                long cost = System.currentTimeMillis() - startTime;
                long waitTime = RATE_LIMIT_INTERVAL - cost;

                if (waitTime > 0) {
                    Thread.sleep(waitTime);
                }

            } catch (InterruptedException e) {
                log.warn("调度线程被中断");
                Thread.currentThread().interrupt();
                break;
            }
        }
    }
}
