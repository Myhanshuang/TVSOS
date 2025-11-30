package com.tvsos.utils;

import java.util.Random;

/**
 * 随机全国城市坐标生成工具（经度在前，纬度在后）
 */
public class MockLocationUtils {

    private static final Random RANDOM = new Random();

    /**
     * 全国主要城市，格式统一为：
     * {longitude, latitude}
     */
    private static final double[][] CITY_POINTS = {
            {116.4074, 39.9042}, // 北京
            {121.4737, 31.2304}, // 上海
            {113.2644, 23.1291}, // 广州
            {114.0579, 22.5431}, // 深圳
            {104.0668, 30.5728}, // 成都
            {106.5516, 29.5630}, // 重庆
            {118.7969, 32.0603}, // 南京
            {108.9398, 34.3416}, // 西安
            {120.1551, 30.2741}, // 杭州
            {120.3826, 36.0671}, // 青岛
            {126.5349, 45.8038}, // 哈尔滨
            {87.6168, 43.8256},  // 乌鲁木齐
            {123.4315, 41.8057}, // 沈阳
            {117.1200, 36.6500}, // 济南
            {102.7123, 25.0307}, // 昆明
            {106.6302, 26.6470}, // 贵阳
            {114.5149, 38.0428}, // 石家庄
            {112.9388, 28.2282}, // 长沙
            {114.3055, 30.5928}  // 武汉
    };


    /**
     * 随机返回城市坐标（±0.5° 偏移）
     * @return [lon, lat]
     */
    public static double[] randomLocation() {
        int idx = RANDOM.nextInt(CITY_POINTS.length);
        double[] base = CITY_POINTS[idx];

        double lon = base[0] + (RANDOM.nextDouble() - 0.5);
        double lat = base[1] + (RANDOM.nextDouble() - 0.5);

        return new double[]{lon, lat};
    }

    public static double[] randomBegin() {
        return randomLocation();
    }

    public static double[] randomEnd() {
        return randomLocation();
    }

    /**
     * 计算两点之间球面距离
     */
    public static double calcDistance(double lon1, double lat1, double lon2, double lat2) {
        double R = 6371.0;

        double dLat = Math.toRadians(lat2 - lat1);
        double dLon = Math.toRadians(lon2 - lon1);

        double a = Math.sin(dLat/2) * Math.sin(dLat/2)
                + Math.cos(Math.toRadians(lat1))
                * Math.cos(Math.toRadians(lat2))
                * Math.sin(dLon/2) * Math.sin(dLon/2);

        double c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
        return R * c;
    }
}
