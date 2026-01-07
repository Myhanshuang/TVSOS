package com.tvsos.service;

import java.util.List;

/**
 * 路径存储服务接口
 * 负责将车辆路径数据持久化到文件系统，避免数据库膨胀
 */
public interface RouteStorageService {

    /**
     * 保存路径到文件
     *
     * @param tripId 行程ID
     * @param segmentIndex 路段索引 (1 或 2)
     * @param points 路径点列表 [lon, lat]
     */
    void saveRoute(Long tripId, Integer segmentIndex, List<Double[]> points);

    /**
     * 从文件加载路径
     *
     * @param tripId 行程ID
     * @param segmentIndex 路段索引
     * @return 路径点列表 [lon, lat]
     */
    List<Double[]> loadRoute(Long tripId, Integer segmentIndex);
    
    /**
     * 检查路径文件是否存在
     */
    boolean exists(Long tripId, Integer segmentIndex);
}
