package com.tvsos.service;

import entity.TransportOrder;

public interface TaskService {
    /**
     * 随机生成一个运输任务并存入数据库
     * 遵循 "符合基本内容" 的逻辑
     * @return 成功生成的 TransportOrder 任务
     */
    TransportOrder createRandomTask();
}