package com.tvsos.service;

public interface DispatchService {
    /**
     * 执行一次调度。
     * 查找所有“待调度”的子任务 (OrderDetail)，并为它们匹配最佳的“空闲”车辆。
     * 具体而言，就是查找数据库中状态为“待调度”的 OrderDetail 记录
     * 然后在数据库中找对应的最匹配的车辆 findBestVehicle
     * 把任务分配给车辆 assignTaskToVehicle，这个过程会更新数据库中的相关记录。
     */
    void dispatchPendingTasks();
}