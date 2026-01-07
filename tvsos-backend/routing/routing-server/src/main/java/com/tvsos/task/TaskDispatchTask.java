package com.tvsos.task;

import com.tvsos.service.TaskService;
import constant.TaskConstant;
import entity.Task;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

import java.util.List;

@Slf4j
@Component
public class TaskDispatchTask {

    @Autowired
    private TaskService taskService;

    @Scheduled(fixedDelayString = "#{T(java.util.concurrent.TimeUnit).MINUTES.toMillis(T(constant.TaskConstant).DISPATCH_MINUTE)}")
    public void dispatchPendingTask(){
        log.info("开始执行任务分配定时任务...(间隔：{} min，最大分配任务：{} 个)", TaskConstant.DISPATCH_MINUTE, TaskConstant.TASK_BATCH_SIZE);

        try {
            // 获取前 batchSize 个待执行任务
            List<Task> tasks = taskService.getPendingTasks(TaskConstant.TASK_BATCH_SIZE);

            if (tasks == null || tasks.isEmpty()) {
                log.info("没有待执行的任务");
                return;
            }

            int successCnt = 0;

            for (Task task : tasks) {
                log.info("分配任务: 任务id-{}", task.getId());
                boolean isSuccess = taskService.dispatchTask(task);
                if(isSuccess){
                    successCnt++;
                }
                if (successCnt == 3) {
                    // 高德 API 有每秒 3 个的并发上限
                    break;
                }
            }

            log.info("本次任务分配完成，总处理任务数: {} (其中成功 {} 次，失败 {} 次)", tasks.size(), successCnt, tasks.size() - successCnt);

        } catch (Exception e) {
            log.error("任务分配任务执行异常", e);
        }
    }
}
