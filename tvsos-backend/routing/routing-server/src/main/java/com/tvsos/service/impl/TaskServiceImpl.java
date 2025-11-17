package com.tvsos.service.impl;

import com.tvsos.mapper.CargoMapper;
import com.tvsos.mapper.ContentCargoMapper; // [!] 引入新 Mapper
import com.tvsos.mapper.PoiMapper;
import com.tvsos.mapper.TransportOrderMapper;
import com.tvsos.service.TaskService;
import constant.StatusConstant;
import dto.PoiQueryDTO;
import entity.Cargo;
import entity.ContentCargo; // [!] 引入新实体
import entity.Poi;
import entity.TransportOrder;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.LocalDateTime;
import java.util.List;
import java.util.Map;
import java.util.Random;
import java.util.UUID;

@Service
@Slf4j
public class TaskServiceImpl implements TaskService {

    //这个表在 cargo_type_mapping 中亦有记载
    //表中货物在cargo_info中亦有记载
    // 货物类型 -> 生产方 POI 类型列表
    private static final Map<Integer, List<Integer>> CARGO_TO_SOURCE_POI_MAP = Map.of(
            1, List.of(1, 2, 3), 2, List.of(4, 17), 3, List.of(4, 6),
            4, List.of(4, 10, 11, 13), 5, List.of(4, 21), 6, List.of(4, 22), 7, List.of(16, 19, 20)
    );
    // 货物类型 -> 消费方 POI 类型列表
    private static final Map<Integer, List<Integer>> CARGO_TO_DEST_POI_MAP = Map.of(
            1, List.of(10, 13, 16, 17), 2, List.of(4, 6), 3, List.of(5, 22, 23),
            4, List.of(10, 11, 12, 13, 14, 15), 5, List.of(5, 23, 25),
            6, List.of(5, 23, 22), 7, List.of(7, 8, 16)
    );

    private final PoiMapper poiMapper;
    private final CargoMapper cargoMapper;
    private final TransportOrderMapper transportOrderMapper;
    private final ContentCargoMapper contentCargoMapper; // [!]
    private final Random random = new Random();

    // 构造函数更新
    public TaskServiceImpl(PoiMapper poiMapper, CargoMapper cargoMapper,
                           TransportOrderMapper transportOrderMapper,
                           ContentCargoMapper contentCargoMapper) { // [!]
        this.poiMapper = poiMapper;
        this.cargoMapper = cargoMapper;
        this.transportOrderMapper = transportOrderMapper;
        this.contentCargoMapper = contentCargoMapper; // [!]
    }

    @Override
    @Transactional
    public TransportOrder createRandomTask() {

        // 1. 智能选择货物和POI (同前)
        int cargoType = random.nextInt(CARGO_TO_SOURCE_POI_MAP.size()) + 1;
        List<Integer> sourcePoiTypes = CARGO_TO_SOURCE_POI_MAP.get(cargoType);
        List<Integer> destPoiTypes = CARGO_TO_DEST_POI_MAP.get(cargoType);
        Cargo cargo = getRandomCargoByType(cargoType);
        Poi beginPoi = getRandomPoiByType(sourcePoiTypes);
        Poi endPoi = getRandomPoiByType(destPoiTypes);

        if (beginPoi == null || endPoi == null || cargo == null) {
            log.warn("【阶段1】创建任务失败：缺少POI或Cargo基础数据 (CargoType: {})", cargoType);
            return null;
        }

        // 2. 随机生成总重量 (10吨上下浮动5吨)
        double totalWeight = 10.0 - 5.0 + (random.nextDouble() * 10.0);
        int quantityInTons = (int) Math.ceil(totalWeight); // 5-15 吨

        // 3. 创建 TransportOrder (总订单)
        TransportOrder order = new TransportOrder();
        order.setOrderNumber(UUID.randomUUID().toString().substring(0, 18));
        order.setBeginPoiId(beginPoi.getId());
        order.setEndPoiId(endPoi.getId());
        order.setCreateTime(LocalDateTime.now());
        order.setStatus(StatusConstant.ORDER_UNPROCESSED); // 0: 待处理 (待拆分)

        transportOrderMapper.insert(order); // 插入后获取 order.getId()

        // 4. [新] 创建 ContentCargo (订单内容)
        ContentCargo content = new ContentCargo();
        content.setTransportOrderId(order.getId()); // [!]
        content.setCargoId(cargo.getId());
        content.setQuantity(quantityInTons); // [!] 存入 5-15 吨的"总重量"
        content.setCreateTime(LocalDateTime.now());

        contentCargoMapper.insert(content);

        log.info("【阶段1】需求生成: 创建总订单 TransportId={}, 货物={}, 总重量={}t",
                order.getId(), cargo.getName(), quantityInTons);

        return order;
    }

    private Poi getRandomPoiByType(List<Integer> types) {
        Integer type = types.get(random.nextInt(types.size()));
        PoiQueryDTO filter = new PoiQueryDTO();
        filter.setTybe(type);
        List<Poi> pois = poiMapper.list(filter);
        if (pois == null || pois.isEmpty()) return null;
        return pois.get(random.nextInt(pois.size()));
    }
    private Cargo getRandomCargoByType(Integer type) {
        List<Cargo> cargos = cargoMapper.findByType(type);
        if (cargos == null || cargos.isEmpty()) return null;
        return cargos.get(random.nextInt(cargos.size()));
    }
}