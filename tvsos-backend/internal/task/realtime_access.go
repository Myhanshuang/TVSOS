package task

import "github.com/kiritosuki/mover/internal/repository"

// VehicleRouteSnapshot 供外部实时接口（WebSocket或大屏）读取的某个车辆正在执行任务路径切片的结构封装。
type VehicleRouteSnapshot struct {
	VehicleID    uint    `json:"vehicleId"`
	ShipmentID   uint    `json:"shipmentId"`
	Sequential   int     `json:"sequential"`
	RouteVersion string  `json:"routeVersion"`
	Progress     int     `json:"progress"`
	Total        int     `json:"total"`
	Points       []Point `json:"points"`
}

// GetVehicleRouteSnapshot 将内部调度的某一时刻的基于路径规划点集的运载进度暴露成只读切片抛出。
// 用于被推送到前端大屏作为实时地理位置回显和路径动态渲染的依据。
func GetVehicleRouteSnapshot(vehicleID uint) (VehicleRouteSnapshot, bool) {
	runningTask, err := repository.GetFirstRunningTaskByVehicle(vehicleID)
	if err != nil || runningTask == nil {
		return VehicleRouteSnapshot{}, false
	}

	meta, points, ok := getRuntimeRouteState(vehicleID)
	if !ok || meta.ShipmentID != runningTask.ShipmentId || meta.Sequential != runningTask.Sequential {
		return VehicleRouteSnapshot{}, false
	}

	return VehicleRouteSnapshot{
		VehicleID:    vehicleID,
		ShipmentID:   runningTask.ShipmentId,
		Sequential:   meta.Sequential,
		RouteVersion: meta.RouteVersion,
		Progress:     meta.Progress,
		Total:        len(points),
		Points:       points,
	}, len(points) > 0
}
