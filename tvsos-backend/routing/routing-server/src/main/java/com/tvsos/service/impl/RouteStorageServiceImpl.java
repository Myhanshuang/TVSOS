package com.tvsos.service.impl;

import com.alibaba.fastjson2.JSON;
import com.alibaba.fastjson2.TypeReference;
import com.tvsos.service.RouteStorageService;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.stereotype.Service;

import java.util.List;
import java.util.concurrent.TimeUnit;

import com.alibaba.fastjson2.JSONArray;
import java.util.ArrayList;

@Slf4j
@Service
public class RouteStorageServiceImpl implements RouteStorageService {

    @Autowired
    private StringRedisTemplate redisTemplate;

    private static final String KEY_PREFIX = "route:";
    private static final long EXPIRE_HOURS = 24;

    private String getKey(Long tripId, Integer segmentIndex) {
        return KEY_PREFIX + tripId + ":" + segmentIndex;
    }

    @Override
    public void saveRoute(Long tripId, Integer segmentIndex, List<Double[]> points) {
        if (points == null || points.isEmpty()) {
            return;
        }
        String key = getKey(tripId, segmentIndex);
        try {
            String jsonContent = JSON.toJSONString(points);
            redisTemplate.opsForValue().set(key, jsonContent, EXPIRE_HOURS, TimeUnit.HOURS);
            log.info("Saved route to Redis: {} (points: {})", key, points.size());
        } catch (Exception e) {
            log.error("Failed to save route to Redis: {}", key, e);
            throw new RuntimeException("Failed to save route data", e);
        }
    }

    @Override
    public List<Double[]> loadRoute(Long tripId, Integer segmentIndex) {
        String key = getKey(tripId, segmentIndex);
        try {
            String jsonContent = redisTemplate.opsForValue().get(key);
            if (jsonContent == null) {
                log.warn("Route not found in Redis: {}", key);
                return null;
            }
            
            // 使用 JSONArray 显式解析，避免泛型数组问题
            JSONArray array = JSON.parseArray(jsonContent);
            if (array == null) return null;
            
            List<Double[]> points = new ArrayList<>();
            for (int i = 0; i < array.size(); i++) {
                JSONArray point = array.getJSONArray(i);
                if (point != null && point.size() >= 2) {
                    points.add(new Double[]{
                        point.getDouble(0),
                        point.getDouble(1)
                    });
                }
            }
            return points;
        } catch (Exception e) {
            log.error("Failed to read route from Redis: {}", key, e);
            return null;
        }
    }

    @Override
    public boolean exists(Long tripId, Integer segmentIndex) {
        String key = getKey(tripId, segmentIndex);
        return Boolean.TRUE.equals(redisTemplate.hasKey(key));
    }
}