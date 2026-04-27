package task

import (
	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/kiritosuki/mover/internal/constant"
	"github.com/kiritosuki/mover/internal/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RecoverRuntimeState 在服务启动时恢复运行时路径状态，避免数据库状态与内存路线不一致。
func RecoverRuntimeState() {
	resetRuntimeRouteCaches()

	tasks, err := repository.GetRunningTasksOrderedByVehicle()
	if err != nil {
		Logger.Logger.Error("启动恢复运行时状态失败：查询运行中任务失败", zap.Error(err))
		return
	}

	runningVehicles := make(map[uint]struct{}, len(tasks))
	for _, task := range tasks {
		if task == nil {
			continue
		}
		runningVehicles[task.VehicleId] = struct{}{}
		if hydrateRuntimeRoute(task.VehicleId, task.ShipmentId, task.Sequential) {
			continue
		}
		if !recoverTaskRoute(task.VehicleId, task.ShipmentId, task.Sequential) {
			Logger.Logger.Warn("启动恢复任务路径失败，将继续后续任务", zap.Uint("vehicleId", task.VehicleId), zap.Uint("shipmentId", task.ShipmentId))
		}
	}

	vehicles, err := repository.ListVehicles(constant.VehicleStatusRunning)
	if err != nil {
		Logger.Logger.Error("启动恢复运行时状态失败：查询运行中车辆失败", zap.Error(err))
		return
	}

	for _, vehicle := range vehicles {
		if vehicle == nil {
			continue
		}
		if _, ok := runningVehicles[vehicle.Id]; ok {
			continue
		}
		reconcileStaleRunningVehicle(vehicle.Id)
	}
}

func hydrateRuntimeRoute(vehicleID uint, shipmentID uint, sequential int) bool {
	meta, points, ok := getRuntimeRouteState(vehicleID)
	if !ok || len(points) == 0 {
		return false
	}
	if meta.ShipmentID != shipmentID || meta.Sequential != sequential {
		return false
	}

	vehicle, err := repository.GetVehicle(int(vehicleID))
	if err == nil {
		alignedMeta := alignRuntimeMetaToPosition(meta, points, vehicle.Lon, vehicle.Lat)
		if alignedMeta.Progress != meta.Progress || alignedMeta.CurrentLon != meta.CurrentLon || alignedMeta.CurrentLat != meta.CurrentLat {
			if updateErr := updateRuntimeRouteMeta(alignedMeta); updateErr != nil {
				Logger.Logger.Warn("恢复 Redis 运行态路径后同步对齐进度失败", zap.Uint("vehicleId", vehicleID), zap.Error(updateErr))
			}
			meta = alignedMeta
		}
	}

	Logger.Logger.Info("从 Redis/内存恢复车辆路径成功", zap.Uint("vehicleId", vehicleID), zap.Uint("shipmentId", shipmentID), zap.Int("progress", meta.Progress), zap.String("routeVersion", meta.RouteVersion))
	return true
}

func recoverTaskRoute(vehicleID uint, shipmentID uint, sequential int) bool {
	vehicle, err := repository.GetVehicle(int(vehicleID))
	if err != nil {
		Logger.Logger.Error("恢复路径失败：查询车辆失败", zap.Uint("vehicleId", vehicleID), zap.Error(err))
		return false
	}

	shipment, err := repository.GetShipment(int(shipmentID))
	if err != nil {
		Logger.Logger.Error("恢复路径失败：查询订单失败", zap.Uint("shipmentId", shipmentID), zap.Error(err))
		return false
	}

	targetPoiID := shipment.StartPoiId
	if sequential == constant.OrderTaskSequentialTransporting {
		targetPoiID = shipment.EndPoiId
	}

	targetPoi, err := repository.GetPoi(int(targetPoiID))
	if err != nil {
		Logger.Logger.Error("恢复路径失败：查询目标POI失败", zap.Uint("shipmentId", shipmentID), zap.Uint("poiId", targetPoiID), zap.Error(err))
		return false
	}

	points := planRouteWithFallback(vehicle.Lon, vehicle.Lat, targetPoi.Lon, targetPoi.Lat)
	if len(points) == 0 {
		Logger.Logger.Warn("恢复路径失败：规划结果为空", zap.Uint("vehicleId", vehicleID), zap.Uint("shipmentId", shipmentID))
		return false
	}

	meta, saveErr := persistRuntimeRoute(vehicleID, shipmentID, sequential, points, 0, vehicle.Lon, vehicle.Lat)
	if saveErr != nil {
		Logger.Logger.Error("恢复路径失败：写入运行态缓存失败", zap.Uint("vehicleId", vehicleID), zap.Uint("shipmentId", shipmentID), zap.Error(saveErr))
		return false
	}

	Logger.Logger.Info("启动恢复车辆路径成功", zap.Uint("vehicleId", vehicleID), zap.Uint("shipmentId", shipmentID), zap.Int("points", len(points)), zap.String("routeVersion", meta.RouteVersion))
	return true
}

func reconcileStaleRunningVehicle(vehicleID uint) {
	plannedTask, err := repository.GetFirstTaskByVehicleAndSequential(vehicleID, constant.OrderTaskSequentialPlanned)
	if err == nil && plannedTask != nil {
		if updateErr := repository.UpdateVehicleStatus(int(vehicleID), constant.VehicleStatusReady, constant.VehicleStatusRunning); updateErr != nil {
			Logger.Logger.Warn("纠正僵尸运行车辆状态失败：running -> ready", zap.Uint("vehicleId", vehicleID), zap.Error(updateErr))
			return
		}
		Logger.Logger.Info("纠正僵尸运行车辆状态：running -> ready", zap.Uint("vehicleId", vehicleID))
		return
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		Logger.Logger.Warn("查询车辆planned任务失败", zap.Uint("vehicleId", vehicleID), zap.Error(err))
	}

	if updateErr := repository.UpdateVehicleStatus(int(vehicleID), constant.VehicleStatusFree, constant.VehicleStatusRunning); updateErr != nil {
		Logger.Logger.Warn("纠正僵尸运行车辆状态失败：running -> free", zap.Uint("vehicleId", vehicleID), zap.Error(updateErr))
		return
	}
	Logger.Logger.Info("纠正僵尸运行车辆状态：running -> free", zap.Uint("vehicleId", vehicleID))
}
