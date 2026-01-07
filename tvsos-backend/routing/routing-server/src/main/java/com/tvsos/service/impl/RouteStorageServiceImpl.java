package com.tvsos.service.impl;

import com.alibaba.fastjson2.JSON;
import com.alibaba.fastjson2.TypeReference;
import com.tvsos.service.RouteStorageService;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;

import java.io.File;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.List;

@Slf4j
@Service
public class RouteStorageServiceImpl implements RouteStorageService {

    // 数据存储根目录，建议配置在 application.yml 中，这里先硬编码为项目下的 data 目录
    // 假设运行目录是 routing-server 或者项目根目录，这里使用相对路径
    private static final String DATA_DIR = "data/routes";

    public RouteStorageServiceImpl() {
        // 确保目录存在
        File dir = new File(DATA_DIR);
        if (!dir.exists()) {
            boolean created = dir.mkdirs();
            if (created) {
                log.info("Created route data directory: {}", dir.getAbsolutePath());
            }
        }
    }

    private String getFileName(Long tripId, Integer segmentIndex) {
        return String.format("%d_%d.json", tripId, segmentIndex);
    }

    @Override
    public void saveRoute(Long tripId, Integer segmentIndex, List<Double[]> points) {
        if (points == null || points.isEmpty()) {
            return;
        }
        String fileName = getFileName(tripId, segmentIndex);
        Path path = Paths.get(DATA_DIR, fileName);
        try {
            String jsonContent = JSON.toJSONString(points);
            Files.write(path, jsonContent.getBytes(StandardCharsets.UTF_8));
            log.info("Saved route to file: {}", path.toAbsolutePath());
        } catch (IOException e) {
            log.error("Failed to save route file: {}", fileName, e);
            throw new RuntimeException("Failed to save route data", e);
        }
    }

    @Override
    public List<Double[]> loadRoute(Long tripId, Integer segmentIndex) {
        String fileName = getFileName(tripId, segmentIndex);
        Path path = Paths.get(DATA_DIR, fileName);
        
        if (!Files.exists(path)) {
            log.warn("Route file not found: {}", path.toAbsolutePath());
            return null;
        }

        try {
            byte[] bytes = Files.readAllBytes(path);
            String jsonContent = new String(bytes, StandardCharsets.UTF_8);
            return JSON.parseObject(jsonContent, new TypeReference<List<Double[]>>(){});
        } catch (IOException e) {
            log.error("Failed to read route file: {}", fileName, e);
            return null;
        }
    }

    @Override
    public boolean exists(Long tripId, Integer segmentIndex) {
        String fileName = getFileName(tripId, segmentIndex);
        Path path = Paths.get(DATA_DIR, fileName);
        return Files.exists(path);
    }
}
