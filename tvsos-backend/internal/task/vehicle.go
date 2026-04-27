package task

import (
	"math"
	"time"

	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/kiritosuki/mover/internal/constant"
	"github.com/kiritosuki/mover/internal/database"
	"github.com/kiritosuki/mover/internal/model"
	"github.com/kiritosuki/mover/internal/realtime"
	"github.com/kiritosuki/mover/internal/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SimulateMoving 模拟车辆行驶
func SimulateMoving() {
	for {
		// 查询正在进行中的任务，按车辆和创建时间排序以保证顺序执行
		tasks, err := repository.GetRunningTasksOrderedByVehicle()
		if err != nil {
			Logger.Logger.Error("查询正在进行中的任务失败")
			time.Sleep(1 * time.Second)
			continue
		}

		// 记录哪些车已经在动了，避免同一辆车在一次循环中启动多个任务
		movingVehicles := make(map[uint]bool)
		for _, task := range tasks {
			if movingVehicles[task.VehicleId] {
				continue
			}
			movingVehicles[task.VehicleId] = true
			go moveVehicle(task)
		}
		speed := GetSimulationSpeed()
		base := time.Duration(constant.VehicleSimulateMovingGap) * time.Second
		interval := time.Duration(float64(base) / speed)
		if interval < 100*time.Millisecond {
			interval = 100 * time.Millisecond
		}
		time.Sleep(interval)
	}
}

// 车辆行驶的核心逻辑
func moveVehicle(task *model.OrderTask) {
	vehicleID := task.VehicleId

	// 发车前的最后防线防脏数据：二次校验车辆状态必须为行驶中
	latestVehicle, err := repository.GetVehicle(int(vehicleID))
	if err != nil || latestVehicle.Status != constant.VehicleStatusRunning {
		Logger.Logger.Warn("数据不一致：任务在途但车辆不处于行驶状态，阻断模拟，需等待系统修正", zap.Uint("vehicleId", vehicleID))
		return
	}

	shipmentID := task.ShipmentId
	meta, points, ok := getRuntimeRouteState(vehicleID)
	if !ok || meta.ShipmentID != shipmentID || meta.Sequential != task.Sequential {
		if !recoverTaskRoute(vehicleID, shipmentID, task.Sequential) {
			return
		}
		meta, points, ok = getRuntimeRouteState(vehicleID)
	}
	if !ok || len(points) == 0 {
		return
	}

	idx := meta.Progress + 1
	// 越界判断 如果这辆车已经走到了这段路的末尾
	if idx >= len(points) {
		handlePathFinished(task)
		return
	}
	// 取当前位置
	point := points[idx]
	meta.Progress = idx
	meta.Total = len(points)
	meta.CurrentLon = point.Lon
	meta.CurrentLat = point.Lat
	meta.UpdatedAt = time.Now().UnixMilli()
	if err := updateRuntimeRouteMeta(meta); err != nil {
		Logger.Logger.Warn("推进车辆运行态进度失败，将继续使用内存状态", zap.Uint("vehicleId", vehicleID), zap.Error(err))
	}

	segmentDistanceKM := 0.0
	if idx > 0 {
		previous := points[idx-1]
		segmentDistanceKM = haversineKM(previous.Lon, previous.Lat, point.Lon, point.Lat)
	}

	emptyMileageDelta := 0.0
	transportDistanceDelta := 0.0
	durationDeltaHour := 0.0
	if segmentDistanceKM > 0 {
		switch task.Sequential {
		case constant.OrderTaskSequentialAccepting:
			emptyMileageDelta = segmentDistanceKM
		case constant.OrderTaskSequentialTransporting:
			transportDistanceDelta = segmentDistanceKM
		}
		durationDeltaHour = estimateDurationHours(segmentDistanceKM, 35)
		_ = repository.AddVehicleTravelMetrics(int(vehicleID), emptyMileageDelta, transportDistanceDelta, durationDeltaHour)
	}

	// 更新数据库中的车辆位置
	err = repository.UpdateVehicleLocation(int(vehicleID), point.Lon, point.Lat)
	if err != nil {
		Logger.Logger.Error("更新车辆位置失败")
		return
	}
	broadcastVehicleEvent("vehicle_move", vehicleID)
	realtime.GlobalHub().BroadcastPath(vehicleID, "progress", buildProgressPayload(meta, point))
}

// 到达路段终点 做状态更新
func handlePathFinished(task *model.OrderTask) {
	switch task.Sequential {
	// 到达起点（准备装货）
	case constant.OrderTaskSequentialAccepting:
		Logger.Logger.Info("车辆到达起点，开始装货", zap.Uint("shipment", task.ShipmentId))
		// 装货（增加载重）
		err := LoadCargo(task)
		if err != nil {
			Logger.Logger.Error("装货失败", zap.Error(err))
		}
		// 更新订单状态：待取货 -> 运输中
		// CAS锁一律格式 旧的status放后面的参数
		err = repository.UpdateShipmentStatus(int(task.ShipmentId), constant.ShipmentStatusWorking, constant.ShipmentStatusWaiting)
		if err != nil {
			Logger.Logger.Error("更新订单状态失败")
			return
		}
		// 更新任务阶段：进入运输
		// 这里同样加了CAS锁(不确定到底加不加 防止go程竞争 加了更保险)
		err = repository.UpdateTaskSequential(int(task.Id), constant.OrderTaskSequentialTransporting, constant.OrderTaskSequentialAccepting)
		if err != nil {
			Logger.Logger.Error("更新任务阶段失败")
			return
		}
		// 重新规划路径（起点 -> 终点）
		shipment, err := repository.GetShipment(int(task.ShipmentId))
		if err != nil {
			Logger.Logger.Error("查询订单失败")
			return
		}
		startPoi, err := repository.GetPoi(int(shipment.StartPoiId))
		if err != nil {
			Logger.Logger.Error("查询起点poi失败")
			return
		}
		endPoi, err := repository.GetPoi((int(shipment.EndPoiId)))
		if err != nil {
			Logger.Logger.Error("查询终点poi失败")
			return
		}
		points := planRouteWithFallback(startPoi.Lon, startPoi.Lat, endPoi.Lon, endPoi.Lat)
		if len(points) == 0 {
			Logger.Logger.Error("重新规划路径失败且兜底为空")
			return
		}
		meta, saveErr := persistRuntimeRoute(task.VehicleId, task.ShipmentId, constant.OrderTaskSequentialTransporting, points, 0, startPoi.Lon, startPoi.Lat)
		if saveErr != nil {
			Logger.Logger.Error("写入运输阶段运行态路径失败", zap.Error(saveErr))
			return
		}
		realtime.GlobalHub().BroadcastPath(task.VehicleId, "full_path", buildFullPathPayload(meta, points))

	// 到达终点（完成任务）
	case constant.OrderTaskSequentialTransporting:
		Logger.Logger.Info("车辆到达终点，任务完成", zap.Uint("shipment", task.ShipmentId))
		// 卸货（减少载重）
		err := UnloadCargo(task)
		if err != nil {
			Logger.Logger.Error("卸货失败", zap.Error(err))
		}

		// 更新任务阶段
		err = repository.UpdateTaskSequential(int(task.Id), constant.OrderTaskSequentialFinish, constant.OrderTaskSequentialTransporting)
		if err != nil {
			Logger.Logger.Error("更新任务阶段为完成失败")
		}

		// 更新订单状态：完成
		err = repository.UpdateShipmentStatus(int(task.ShipmentId), constant.ShipmentStatusFinish, constant.ShipmentStatusWorking)
		if err != nil {
			Logger.Logger.Error("更新订单状态为完成失败")
		}

		meta, _, _ := getRuntimeRouteState(task.VehicleId)
		clearRuntimeRoute(task.VehicleId)
		realtime.GlobalHub().BroadcastPath(task.VehicleId, "clear", buildClearPayload(meta))

		// 如果该车还有已分配待发车任务，立即拉起下一单；否则释放车辆。
		if !startNextPlannedTaskForVehicle(task.VehicleId) {
			_ = repository.UpdateVehicleStatus(int(task.VehicleId), constant.VehicleStatusFree, constant.VehicleStatusRunning)
			broadcastVehicleEvent("vehicle_update", task.VehicleId)
		}
	}
}

func broadcastVehicleEvent(event string, vehicleID uint) {
	latestVehicle, getErr := repository.GetVehicle(int(vehicleID))
	if getErr != nil {
		return
	}
	realtime.GlobalHub().BroadcastVehicles(event, repository.BuildVehicleView(latestVehicle))
}

func haversineKM(lon1, lat1, lon2, lat2 float64) float64 {
	const earthRadiusKm = 6371.0
	toRadians := func(deg float64) float64 {
		return deg * math.Pi / 180
	}

	dlon := toRadians(lon2 - lon1)
	dlat := toRadians(lat2 - lat1)
	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(toRadians(lat1))*math.Cos(toRadians(lat2))*
			math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

func estimateDurationHours(distanceKM float64, speedKmh float64) float64 {
	if speedKmh <= 0 {
		return 0
	}
	return distanceKM / speedKmh
}

// LoadCargo 装货逻辑：增加当前车辆的载重
func LoadCargo(task *model.OrderTask) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var shipment model.Shipment
		if err := tx.First(&shipment, task.ShipmentId).Error; err != nil {
			return err
		}
		var cargo model.Cargo
		if err := tx.First(&cargo, shipment.CargoId).Error; err != nil {
			return err
		}
		var vehicle model.Vehicle
		if err := tx.First(&vehicle, task.VehicleId).Error; err != nil {
			return err
		}

		totalWeight := shipment.Count * cargo.Weight
		newReserved := vehicle.ReservedSize - totalWeight
		if newReserved < 0 {
			newReserved = 0
		}
		return tx.Model(&model.Vehicle{}).
			Where("id = ?", task.VehicleId).
			Updates(map[string]interface{}{
				"size":          vehicle.Size + totalWeight,
				"reserved_size": newReserved,
			}).Error
	})
}

// UnloadCargo 卸货逻辑：扣减当前车辆的载重
func UnloadCargo(task *model.OrderTask) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var shipment model.Shipment
		if err := tx.First(&shipment, task.ShipmentId).Error; err != nil {
			return err
		}
		var cargo model.Cargo
		if err := tx.First(&cargo, shipment.CargoId).Error; err != nil {
			return err
		}
		var vehicle model.Vehicle
		if err := tx.First(&vehicle, task.VehicleId).Error; err != nil {
			return err
		}

		totalWeight := shipment.Count * cargo.Weight
		newSize := vehicle.Size - totalWeight
		if newSize < 0 {
			newSize = 0
		}
		return tx.Model(&model.Vehicle{}).Where("id = ?", task.VehicleId).Update("size", newSize).Error
	})
}
