package com.tvsos.utils;

import com.alibaba.fastjson2.JSONArray;
import com.alibaba.fastjson2.JSONObject;
import exception.ServiceException;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

@Slf4j
@Component
public class TripUtils {
    @Value("${amap.key}")
    private String amapKey;
    // 高德驾车路径接口（v5/v3 两者略有差别，使用常见字段解析）
    private String AMAP_ROUTING_URL = "https://restapi.amap.com/v3/direction/driving";

    /**
     * 规划路线
     * @param origin
     * @param destination
     * @param waypoints
     * @return
     */
    public Map<String, Object> planTrip(String origin, String destination, String waypoints){
        Map<String, String> params = new HashMap<>();
        params.put("key", amapKey);
        params.put("origin", origin);
        params.put("destination", destination);
        if (waypoints != null && !waypoints.isEmpty()) {
            params.put("waypoints", waypoints);
        }
        params.put("extensions", "all");
        // 默认策略可保留

        String body = HttpUtils.doGet(AMAP_ROUTING_URL, params);
        if (body == null || body.isEmpty()) {
            throw new ServiceException("高德返回空响应");
        }

        JSONObject root = JSONObject.parseObject(body);
        // 高德 v3 返回 status=1 成功
        String status = root.getString("status");
        if (!"1".equals(status)) {
            String info = root.getString("info");
            log.warn("高德路径规划失败 status={} info={} body={}", status, info, body);
            throw new ServiceException("高德路径规划失败: " + info);
        }

        JSONObject route = root.getJSONObject("route");
        if (route == null) {
            throw new ServiceException("高德返回无 route");
        }

        JSONArray paths = route.getJSONArray("paths");
        if (paths == null || paths.isEmpty()) {
            throw new ServiceException("高德返回无 paths");
        }

        JSONObject p0 = paths.getJSONObject(0);
        double distance = p0.getDoubleValue("distance") / 1000; // km
        int duration = p0.getIntValue("duration") / 3600; // h
        JSONArray steps = p0.getJSONArray("steps");

        // 抽取 polyline（整条路线的点，转换为 List<double[]>）
        List<double[]> polyline = new ArrayList<>();

        double[] lastPoint = null;

        for (int i = 0; i < steps.size(); i++) {
            JSONObject step = steps.getJSONObject(i);
            String seg = step.getString("polyline"); // "lon,lat;lon,lat;..."

            if (seg == null || seg.isEmpty()){
                continue;
            }

            // 切分该 step 的所有点
            String[] pointPairs = seg.split(";");
            for (String pair : pointPairs) {
                if (pair == null || pair.isEmpty()){
                    continue;
                }

                String[] xy = pair.split(",");
                if (xy.length != 2){
                    continue;
                }

                double lon;
                double lat;
                try {
                    lon = Double.parseDouble(xy[0]);
                    lat = Double.parseDouble(xy[1]);
                } catch (NumberFormatException e) {
                    continue;
                }

                double[] point = new double[]{lon, lat};

                // 去掉连续重复点（step 的末尾经常和下一 step 的开头重复）
                if (lastPoint == null || lastPoint[0] != lon || lastPoint[1] != lat) {
                    polyline.add(point);
                    lastPoint = point;
                }
            }
        }

        Map<String, Object> out = new HashMap<>();
        out.put("distance", distance);
        out.put("duration", duration);
        out.put("polyline", polyline);
        out.put("steps", steps);
        out.put("raw", root);
        return out;
    }
}
