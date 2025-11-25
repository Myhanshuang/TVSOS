package com.tvsos.service;

import entity.Task;

import java.util.List;

public interface TaskService {
    /**
     * 获取待分配的任务 前 taskBatchSize 个
     * @param taskBatchSize
     * @return
     */
    List<Task> getPendingTasks(Integer taskBatchSize);

    /**
     * 分配任务
     * @param task
     */
    boolean dispatchTask(Task task);
}
