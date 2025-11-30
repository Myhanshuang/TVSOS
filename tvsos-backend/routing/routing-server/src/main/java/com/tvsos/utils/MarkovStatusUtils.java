// ========== 代码开始： MarkovStateUtil.java ==========
package com.tvsos.utils;

import java.util.Random;

/**
 * 马尔可夫链状态转移工具
 *
 * 说明：
 * - 状态编号与含义（与你的约定一致）：
 *   车辆状态 1空闲 2接单行驶 3装货 4运货行驶 5卸货中 6停留等待 7加油 8维修
 * - 每次完成一个阶段性任务（例如到达一个 task 的终点并完成该 task）时调用 nextState
 * - 该类不做数据库操作，仅返回下一个状态，数据库写入由调用方完成
 */
public class MarkovStatusUtils {

    private static final Random RNG = new Random();

    /**
     * 简化版马尔可夫链概率矩阵（按我们优化后的版本）
     * rows = current state index (0-based for state 1..8)
     * cols = next state probabilities for states 1..8
     *
     * 意义：PROBS[current-1][next-1]
     */
    private static final double[][] PROBS = new double[][] {
            // 1 空闲：不由 Markov 转移，保持自己
            {1.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0},

            // 2 接单行驶 -> 3 装货
            {0.0, 0.0, 1.0, 0.0, 0.0, 0.0, 0.0, 0.0},

            // 3 装货 -> 4 运货行驶
            {0.0, 0.0, 0.0, 1.0, 0.0, 0.0, 0.0, 0.0},

            // 4 运货行驶 -> 5 卸货
            {0.0, 0.0, 0.0, 0.0, 1.0, 0.0, 0.0, 0.0},

            // 5 卸货 -> 1 空闲
            {1.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0},

            // 6 停留等待（不使用，直接保持自身）
            {0.0, 0.0, 0.0, 0.0, 0.0, 1.0, 0.0, 0.0},

            // 7 加油 -> 自动回到空闲
            {1.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0},

            // 8 维修 -> 自动回到空闲
            {1.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0}
    };

    /**
     * 根据当前状态（1..8）返回下一个状态（1..8）
     * 如果输入不合法，返回当前状态（不变）
     */
    public static int nextState(int currentState) {
        if (currentState < 1 || currentState > 8) return currentState;
        double[] row = PROBS[currentState - 1];
        double r = RNG.nextDouble();
        double cum = 0.0;
        for (int i = 0; i < row.length; i++) {
            cum += row[i];
            if (r <= cum) {
                return i + 1;
            }
        }
        // 万一浮点累计误差，返回最后一个有概率的状态或当前
        for (int i = row.length - 1; i >= 0; i--) {
            if (row[i] > 0) return i + 1;
        }
        return currentState;
    }

    /**
     * 可选：暴露概率矩阵（便于单元测试）
     */
    public static double[][] getProbabilityMatrix() {
        return PROBS;
    }
}
// ========== 代码结束 ==========
