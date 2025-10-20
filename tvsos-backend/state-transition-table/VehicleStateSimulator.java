// package com.tvsos.xxx; 根据实际项目结构调整包名

import java.util.HashMap;
import java.util.Map;
import java.util.Random;

/**
 * 货车状态枚举
 */
enum TruckState {

    DRIVING_TO_PICKUP("接单行驶"),
    LOADING("装货"),
    DELIVERING("运货行驶"),
    UNLOADING("卸货"),
    WAITING_ON_SITE("停留等待"),
    IDLE("空闲"),
    REFUELING("加油"),
    MAINTENANCE("维修保养");

    private final String description;

    TruckState(String description) {
        this.description = description;
    }

    @Override
    public String toString() {
        return this.description;
    }
}

public class VehicleStateSimulator {

    // 定义状态转移矩阵
    // Key: 当前状态
    // Value: 一个Map，其中 Key 是下一个可能的状态, Value 是转移到该状态的概率
    //这个三元组结构表示每个状态到其他状态的转移概率
    //使用随机数生成器来模拟状态转移
    private static final Map<TruckState, Map<TruckState, Double>> transitionMatrix = new HashMap<>();
    private static final Random random = new Random();

    // 使用静态代码块在类加载时初始化概率表
    static {
        // 状态 1: 接单行驶
        Map<TruckState, Double> fromDrivingToPickup = new HashMap<>();
        fromDrivingToPickup.put(TruckState.LOADING, 0.90);
        fromDrivingToPickup.put(TruckState.WAITING_ON_SITE, 0.03);
        fromDrivingToPickup.put(TruckState.REFUELING, 0.05);
        fromDrivingToPickup.put(TruckState.MAINTENANCE, 0.02);
        transitionMatrix.put(TruckState.DRIVING_TO_PICKUP, fromDrivingToPickup);

        // 状态 2: 装货
        Map<TruckState, Double> fromLoading = new HashMap<>();
        fromLoading.put(TruckState.DELIVERING, 0.98);
        fromLoading.put(TruckState.WAITING_ON_SITE, 0.02);
        transitionMatrix.put(TruckState.LOADING, fromLoading);

        // 状态 3: 运货行驶
        Map<TruckState, Double> fromDelivering = new HashMap<>();
        fromDelivering.put(TruckState.UNLOADING, 0.90);
        fromDelivering.put(TruckState.REFUELING, 0.07);
        fromDelivering.put(TruckState.MAINTENANCE, 0.03);
        transitionMatrix.put(TruckState.DELIVERING, fromDelivering);

        // 状态 4: 卸货
        Map<TruckState, Double> fromUnloading = new HashMap<>();
        fromUnloading.put(TruckState.DRIVING_TO_PICKUP, 0.30);
        fromUnloading.put(TruckState.WAITING_ON_SITE, 0.15);
        fromUnloading.put(TruckState.IDLE, 0.40);
        fromUnloading.put(TruckState.REFUELING, 0.10);
        fromUnloading.put(TruckState.MAINTENANCE, 0.05);
        transitionMatrix.put(TruckState.UNLOADING, fromUnloading);

        // 状态 5: 停留等待
        Map<TruckState, Double> fromWaiting = new HashMap<>();
        fromWaiting.put(TruckState.DRIVING_TO_PICKUP, 0.70);
        fromWaiting.put(TruckState.IDLE, 0.20);
        fromWaiting.put(TruckState.REFUELING, 0.05);
        fromWaiting.put(TruckState.MAINTENANCE, 0.05);
        transitionMatrix.put(TruckState.WAITING_ON_SITE, fromWaiting);

        // 状态 6: 空闲
        Map<TruckState, Double> fromIdle = new HashMap<>();
        fromIdle.put(TruckState.DRIVING_TO_PICKUP, 0.80);
        fromIdle.put(TruckState.WAITING_ON_SITE, 0.10);
        fromIdle.put(TruckState.REFUELING, 0.05);
        fromIdle.put(TruckState.MAINTENANCE, 0.05);
        transitionMatrix.put(TruckState.IDLE, fromIdle);

        // 状态 7: 加油
        Map<TruckState, Double> fromRefueling = new HashMap<>();
        fromRefueling.put(TruckState.DRIVING_TO_PICKUP, 0.30);
        fromRefueling.put(TruckState.WAITING_ON_SITE, 0.10);
        fromRefueling.put(TruckState.IDLE, 0.60);
        transitionMatrix.put(TruckState.REFUELING, fromRefueling);

        // 状态 8: 维修保养
        Map<TruckState, Double> fromMaintenance = new HashMap<>();
        fromMaintenance.put(TruckState.WAITING_ON_SITE, 0.10);
        fromMaintenance.put(TruckState.IDLE, 0.90);
        transitionMatrix.put(TruckState.MAINTENANCE, fromMaintenance);
    }

    /**
     * 根据当前状态和预设的概率表，计算并返回货车的下一个状态。
     * @param currentState 货车的当前状态
     * @return 货车的下一个状态
     */

    public TruckState getNextState(TruckState currentState) {
        // 获取当前状态对应的所有可能转移
        Map<TruckState, Double> transitions = transitionMatrix.get(currentState);
        if (transitions == null || transitions.isEmpty()) {
            // 如果某个状态在表中没有定义转移，则默认保持原状态
            return currentState;
        }

        // 生成一个0.0到1.0之间的随机数
        double randomValue = random.nextDouble();

        // 遍历可能的下一个状态，构建概率区间并判断随机数落在哪个区间
        double cumulativeProbability = 0.0;
        for (Map.Entry<TruckState, Double> entry : transitions.entrySet()) {
            cumulativeProbability += entry.getValue();
            if (randomValue < cumulativeProbability) {
                // 随机数落在了当前状态的概率区间内，返回该状态
                return entry.getKey();
            }
        }

        // 如果概率总和为1.0，代码不会执行到这里。
        // 这是一个备用逻辑，以防浮点数精度问题导致 randomValue 恰好是1.0。
        // 这种情况下，可以返回最后一个状态或抛出异常。
        // 为确保健壮性，直接返回当前状态。
        return currentState;
    }
}