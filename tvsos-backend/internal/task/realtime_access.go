package task

import "github.com/kiritosuki/mover/internal/repository"

type VehicleRouteSnapshot struct {
	VehicleID    uint    `json:"vehicleId"`
	ShipmentID   uint    `json:"shipmentId"`
	Sequential   int     `json:"sequential"`
	RouteVersion string  `json:"routeVersion"`
	Progress     int     `json:"progress"`
	Total        int     `json:"total"`
	Points       []Point `json:"points"`
}

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
