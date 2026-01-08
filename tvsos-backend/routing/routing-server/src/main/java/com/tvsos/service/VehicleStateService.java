package com.tvsos.service;

import dto.ModbusVehicleDTO;
import dto.ModbusFrontendDTO;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;

import java.time.Instant;
import java.time.LocalDateTime;
import java.time.ZoneId;
import java.util.Collection;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

/**
 * 内存车辆状态服务
 * 使用 ConcurrentHashMap 来线程安全地存储和更新车辆的最新位置信息
 */
@Slf4j
@Service
public class VehicleStateService {

    /**
     * 核心存储：一个线程安全的 Map。
     * Key: 车牌号 (String)
     * Value: 车辆的最新状态 (VehicleStateDTO)
     */
    private final Map<String, ModbusFrontendDTO> latestVehicleStates = new ConcurrentHashMap<>();

    /**
     * 当接收到新的位置上报时，更新内存中的状态
     * @param reportDTO 从 Python 脚本接收的数据
     */
    public void updateVehicleState(ModbusVehicleDTO reportDTO) {
        // 创建或更新车辆状态对象
        ModbusFrontendDTO state = new ModbusFrontendDTO();
        state.setLicensePlate(reportDTO.getLicense());
        state.setLatitude(reportDTO.getLat());
        state.setLongitude(reportDTO.getLon());
        state.setSpeed(reportDTO.getSpeed());

        // 将新状态放入 Map 中，如果已存在同车牌号的记录，则会直接覆盖
        latestVehicleStates.put(state.getLicensePlate(), state);
        log.info("已更新车辆 '{}' 的内存状态。", state.getLicensePlate());
    }

    /**
     * 获取当前内存中所有车辆的最新状态
     * @return 车辆状态的集合
     */
    public Collection<ModbusFrontendDTO> getAllLatestStates() {
        return latestVehicleStates.values();
    }
}
