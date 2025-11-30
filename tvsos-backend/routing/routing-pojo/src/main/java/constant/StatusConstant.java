package constant;

public class StatusConstant {
    public static final Integer DRIVING_TO_PICKUP = 1;       // 接单行驶
    public static final Integer LOADING = 2;                 // 装货
    public static final Integer DRIVING_TO_DELIVER = 3;      // 运货行驶
    public static final Integer UNLOADING = 4;               // 卸货
    public static final Integer WAITING = 5;                 // 停留等待
    public static final Integer FREE = 6;                    // 空闲
    public static final Integer REFUELING = 7;               // 加油
    public static final Integer MAINTENANCE = 8;             // 维修保养
    public static final Integer BREAKDOWN = 9;               // 故障

    // [新] TransportOrder (总订单) 状态
    public static final Integer ORDER_UNPROCESSED = 0; // 待处理 (待拆分)
    public static final Integer ORDER_PROCESSED = 1;   // 已处理 (已拆分)
    public static final Integer ORDER_COMPLETED = 4;   // 已完成

    // [新] OrderDetail (子任务) 状态
    public static final Integer TASK_READY_FOR_DISPATCH = 1; // 待调度
    public static final Integer TASK_ASSIGNED = 2;           // 已分配 (待执行)
    public static final Integer TASK_IN_TRANSIT = 3;         // 运输中
    public static final Integer TASK_COMPLETED = 4;          // 已完成
}
