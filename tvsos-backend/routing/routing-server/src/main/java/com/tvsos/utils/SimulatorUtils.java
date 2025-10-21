package com.tvsos.utils;

import constant.StatusConstant;

import java.util.*;

public class SimulatorUtils {
    private static final Random RANDOM = new Random();

    /**
     * 状态转移概率表（马尔可夫链）
     * 每个状态对应一个 TreeMap：key 为累积概率上限，value 为下一个状态
     */
    private static final Map<Integer, NavigableMap<Double, Integer>> transitionMap = new HashMap<>();

    static {
        // 1. 接单行驶
        addTransitions(StatusConstant.DRIVING_TO_PICKUP, new double[]{
                0.88, 0.91, 0.96, 0.98, 1.00
        }, new int[]{
                StatusConstant.LOADING,
                StatusConstant.WAITING,
                StatusConstant.REFUELING,
                StatusConstant.MAINTENANCE,
                StatusConstant.BREAKDOWN
        });

        // 2. 装货
        addTransitions(StatusConstant.LOADING, new double[]{
                0.97, 0.99, 1.00
        }, new int[]{
                StatusConstant.DRIVING_TO_DELIVER,
                StatusConstant.WAITING,
                StatusConstant.BREAKDOWN
        });

        // 3. 运货行驶
        addTransitions(StatusConstant.DRIVING_TO_DELIVER, new double[]{
                0.87, 0.94, 0.97, 1.00
        }, new int[]{
                StatusConstant.UNLOADING,
                StatusConstant.REFUELING,
                StatusConstant.MAINTENANCE,
                StatusConstant.BREAKDOWN
        });

        // 4. 卸货
        addTransitions(StatusConstant.UNLOADING, new double[]{
                0.30, 0.45, 0.84, 0.94, 0.99, 1.00
        }, new int[]{
                StatusConstant.DRIVING_TO_PICKUP,
                StatusConstant.WAITING,
                StatusConstant.FREE,
                StatusConstant.REFUELING,
                StatusConstant.MAINTENANCE,
                StatusConstant.BREAKDOWN
        });

        // 5. 停留等待
        addTransitions(StatusConstant.WAITING, new double[]{
                0.70, 0.89, 0.94, 0.99, 1.00
        }, new int[]{
                StatusConstant.DRIVING_TO_PICKUP,
                StatusConstant.FREE,
                StatusConstant.REFUELING,
                StatusConstant.MAINTENANCE,
                StatusConstant.BREAKDOWN
        });

        // 6. 空闲
        addTransitions(StatusConstant.FREE, new double[]{
                0.79, 0.89, 0.94, 0.99, 1.00
        }, new int[]{
                StatusConstant.DRIVING_TO_PICKUP,
                StatusConstant.WAITING,
                StatusConstant.REFUELING,
                StatusConstant.MAINTENANCE,
                StatusConstant.BREAKDOWN
        });

        // 7. 加油
        addTransitions(StatusConstant.REFUELING, new double[]{
                0.30, 0.40, 0.99, 1.00
        }, new int[]{
                StatusConstant.DRIVING_TO_PICKUP,
                StatusConstant.WAITING,
                StatusConstant.FREE,
                StatusConstant.BREAKDOWN
        });

        // 8. 维修保养
        addTransitions(StatusConstant.MAINTENANCE, new double[]{
                0.10, 1.00
        }, new int[]{
                StatusConstant.WAITING,
                StatusConstant.FREE
        });

        // 9. 故障
        addTransitions(StatusConstant.BREAKDOWN, new double[]{
                1.00
        }, new int[]{
                StatusConstant.MAINTENANCE
        });
    }

    /**
     * 添加状态转移概率（累积形式）
     */
    private static void addTransitions(int currentState, double[] probs, int[] nextStates) {
        if (probs.length != nextStates.length) {
            throw new IllegalArgumentException("概率表与状态表长度不一致");
        }
        NavigableMap<Double, Integer> map = new TreeMap<>();
        for (int i = 0; i < probs.length; i++) {
            map.put(probs[i], nextStates[i]);
        }
        transitionMap.put(currentState, map);
    }

    /**
     * 根据当前状态模拟下一个状态（马尔可夫状态转移）
     *
     * @param currentStatus 当前状态（Integer）
     * @return 下一个状态（Integer）
     */
    public static Integer simulateStatus(Integer currentStatus) {
        NavigableMap<Double, Integer> map = transitionMap.get(currentStatus);
        if (map == null) {
            throw new IllegalArgumentException("未知状态: " + currentStatus);
        }

        double randomValue = RANDOM.nextDouble(); // [0, 1)
        for (Map.Entry<Double, Integer> entry : map.entrySet()) {
            if (randomValue <= entry.getKey()) {
                return entry.getValue();
            }
        }
        // 理论上不会走到这里
        return currentStatus;
    }
}