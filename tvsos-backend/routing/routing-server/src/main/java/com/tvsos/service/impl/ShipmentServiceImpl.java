package com.tvsos.service.impl;

import com.tvsos.mapper.CargoMapper;
import com.tvsos.mapper.ShipmentCargoMapper;
import com.tvsos.mapper.ShipmentMapper;
import com.tvsos.mapper.TaskMapper;
import com.tvsos.service.ShipmentService;
import com.tvsos.utils.MockLocationUtils;
import entity.Cargo;
import entity.Shipment;
import entity.ShipmentCargo;
import entity.Task;
import exception.ServiceException;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.List;
import java.util.Random;
import java.util.UUID;

@Slf4j
@Service
public class ShipmentServiceImpl implements ShipmentService {

    @Autowired
    private ShipmentMapper shipmentMapper;

    @Autowired
    private ShipmentCargoMapper shipmentCargoMapper;

    @Autowired
    private CargoMapper cargoMapper;

    @Autowired
    private TaskMapper taskMapper;

    private static final Random RANDOM = new Random();

    /**
     * mock用户订单
     *
     * @param count
     * @return
     */
    @Override
    @Transactional
    public List<Shipment> mockShipments(int count) {
        List<Shipment> list = new ArrayList<>();

        // 读取 cargo 表所有货物
        List<Cargo> cargoList = cargoMapper.findAll();
        if (cargoList.isEmpty()) {
            throw new ServiceException("cargo 表没有数据，无法 mock 订单");
        }

        for (int i = 0; i < count; i++) {

            Shipment shipment = new Shipment();
            String uuid = UUID.randomUUID().toString().replace("-", "").substring(0, 16);
            String num = "SHP-" + uuid;
            shipment.setNum(num);

            // ----------------------------
            // 经纬度随机（lon,lat）
            // ----------------------------
            double[] begin = MockLocationUtils.randomBegin();
            double[] end = MockLocationUtils.randomEnd();

            shipment.setBeginLon(begin[0]);
            shipment.setBeginLat(begin[1]);
            shipment.setEndLon(end[0]);
            shipment.setEndLat(end[1]);

            // ----------------------------
            // 根据距离估算预计时间
            // ----------------------------
            double distanceKm = MockLocationUtils.calcDistance(begin[0], begin[1], end[0], end[1]);

            // 假设速度 40~60 km/h
            double speed = 40 + RANDOM.nextInt(20);
            double hours = distanceKm / speed;

            LocalDateTime now = LocalDateTime.now();

            shipment.setEstBeginTime(now.plusHours(RANDOM.nextInt(2)));
            shipment.setEstEndTime(
                    shipment.getEstBeginTime().plusMinutes((long) (hours * 60))
            );

            shipment.setCreateTime(now);
            shipment.setStatus(1); // 待拆分

            // === 保存订单 ===
            shipmentMapper.insert(shipment);

            // ----------------------------
            // Mock 货物（随机 1~5 种）
            // ----------------------------
            int cargoTypes = 1 + RANDOM.nextInt(5);

            for (int c = 0; c < cargoTypes; c++) {

                Cargo cargo = cargoList.get(RANDOM.nextInt(cargoList.size()));

                int quantity = 1 + RANDOM.nextInt(500);

                ShipmentCargo sc = new ShipmentCargo();
                sc.setShipmentId(shipment.getId());
                sc.setCargoId(cargo.getId());
                sc.setQuantity(quantity);

                // 关键点：重量 = 每件货物重量 × 数量
                sc.setWeight(cargo.getWeight() * quantity);

                shipmentCargoMapper.insert(sc);
            }

            list.add(shipment);
        }

        return list;
    }

    /**
     * 获取即将拆分的用户订单 前 batchSize 个
     *
     * @param batchSize
     * @return
     */
    @Override
    public List<Shipment> getPendingShipments(int batchSize) {
        return shipmentMapper.getPendingShipments(batchSize);
    }

    /**
     * 拆分用户订单
     *
     * @param shipment    待拆分订单
     * @param weightLimit 单个任务货物重量上限
     */
    @Override
    public void splitShipment(Shipment shipment, double weightLimit) {
        List<ShipmentCargo> cargos = shipmentCargoMapper.getByShipmentId(shipment.getId());
        if (cargos == null || cargos.isEmpty()) {
            log.warn("订单 {} 无货物信息，跳过拆分", shipment.getNum());
            return;
        }

        for (ShipmentCargo cargo : cargos) {
            double totalWeight = cargo.getWeight();
            int quantity = cargo.getQuantity();
            double singleWeight = totalWeight / quantity;

            int remainQuantity = quantity;

            while (remainQuantity > 0) {
                int taskQuantity;
                double taskWeight;

                // 如果单个货物重量就大于限制，则一个任务放1件
                if (singleWeight >= weightLimit) {
                    taskQuantity = 1;
                    taskWeight = singleWeight;
                } else {
                    // 根据上限计算本次任务数量
                    int maxQuantity = (int) Math.floor(weightLimit / singleWeight);
                    taskQuantity = Math.min(remainQuantity, maxQuantity);
                    taskWeight = taskQuantity * singleWeight;
                }

                Task task = new Task();
                task.setShipmentId(shipment.getId());
                task.setCargoId(cargo.getCargoId());
                task.setQuantity(taskQuantity);
                task.setWeight(taskWeight);
                task.setBeginLon(shipment.getBeginLon());
                task.setBeginLat(shipment.getBeginLat());
                task.setEndLon(shipment.getEndLon());
                task.setEndLat(shipment.getEndLat());
                task.setStatus(1); // 待调度

                taskMapper.insert(task);

                remainQuantity -= taskQuantity;
            }
        }

        // 更新订单状态为已拆分（2:待调度）
        shipment.setStatus(2);
        shipmentMapper.update(shipment);

        // TODO: 后续调用高德地图API，将任务分配给车辆
    }
}

