package task

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/kiritosuki/mover/internal/config"
	"github.com/kiritosuki/mover/internal/database"
	goRedis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const runtimeRouteAlignmentThresholdKM = 0.1

type RuntimeRouteMeta struct {
	VehicleID    uint    `json:"vehicleId"`
	ShipmentID   uint    `json:"shipmentId"`
	Sequential   int     `json:"sequential"`
	RouteVersion string  `json:"routeVersion"`
	Progress     int     `json:"progress"`
	Total        int     `json:"total"`
	CurrentLon   float64 `json:"currentLon"`
	CurrentLat   float64 `json:"currentLat"`
	UpdatedAt    int64   `json:"updatedAt"`
}

type runtimeRoutePathCache struct {
	mu        sync.RWMutex
	byVersion map[string][]Point
}

type runtimeRouteMetaCache struct {
	mu        sync.RWMutex
	byVehicle map[uint]RuntimeRouteMeta
}

var routePathCache = runtimeRoutePathCache{byVersion: make(map[string][]Point)}
var routeMetaCache = runtimeRouteMetaCache{byVehicle: make(map[uint]RuntimeRouteMeta)}

func resetRuntimeRouteCaches() {
	routePathCache.mu.Lock()
	routePathCache.byVersion = make(map[string][]Point)
	routePathCache.mu.Unlock()

	routeMetaCache.mu.Lock()
	routeMetaCache.byVehicle = make(map[uint]RuntimeRouteMeta)
	routeMetaCache.mu.Unlock()
}

func runtimeRouteKeyPrefix() string {
	prefix := strings.TrimSpace(config.VP.GetString("redis.key_prefix"))
	if prefix == "" {
		return "mover"
	}
	return prefix
}

func runtimeRouteTTL() time.Duration {
	hours := config.VP.GetInt("redis.route_ttl_hours")
	if hours <= 0 {
		hours = 6
	}
	return time.Duration(hours) * time.Hour
}

func runtimeRoutePathKey(routeVersion string) string {
	return fmt.Sprintf("%s:route:path:%s", runtimeRouteKeyPrefix(), routeVersion)
}

func runtimeRouteVehicleKey(vehicleID uint) string {
	return fmt.Sprintf("%s:route:vehicle:%d", runtimeRouteKeyPrefix(), vehicleID)
}

func newRouteVersion(vehicleID uint, shipmentID uint, sequential int) string {
	return fmt.Sprintf("v%d-s%d-q%d-%d", vehicleID, shipmentID, sequential, time.Now().UnixMilli())
}

func clampProgress(progress int, total int) int {
	if total <= 0 {
		return 0
	}
	if progress < 0 {
		return 0
	}
	if progress >= total {
		return total - 1
	}
	return progress
}

func copyPoints(points []Point) []Point {
	cloned := make([]Point, len(points))
	copy(cloned, points)
	return cloned
}

func cacheRoutePoints(routeVersion string, points []Point) {
	routePathCache.mu.Lock()
	routePathCache.byVersion[routeVersion] = copyPoints(points)
	routePathCache.mu.Unlock()
}

func cacheRouteMeta(meta RuntimeRouteMeta) {
	routeMetaCache.mu.Lock()
	routeMetaCache.byVehicle[meta.VehicleID] = meta
	routeMetaCache.mu.Unlock()
}

func dropRouteVersion(routeVersion string) {
	routePathCache.mu.Lock()
	delete(routePathCache.byVersion, routeVersion)
	routePathCache.mu.Unlock()
}

func dropVehicleMeta(vehicleID uint) {
	routeMetaCache.mu.Lock()
	delete(routeMetaCache.byVehicle, vehicleID)
	routeMetaCache.mu.Unlock()
}

func buildRuntimeRouteMeta(vehicleID uint, shipmentID uint, sequential int, routeVersion string, points []Point, progress int, currentLon float64, currentLat float64) RuntimeRouteMeta {
	meta := RuntimeRouteMeta{
		VehicleID:    vehicleID,
		ShipmentID:   shipmentID,
		Sequential:   sequential,
		RouteVersion: routeVersion,
		Total:        len(points),
		Progress:     clampProgress(progress, len(points)),
		CurrentLon:   currentLon,
		CurrentLat:   currentLat,
		UpdatedAt:    time.Now().UnixMilli(),
	}
	if len(points) > 0 {
		anchor := points[meta.Progress]
		if meta.CurrentLon == 0 && meta.CurrentLat == 0 {
			meta.CurrentLon = anchor.Lon
			meta.CurrentLat = anchor.Lat
		}
	}
	return meta
}

func persistRuntimeRoute(vehicleID uint, shipmentID uint, sequential int, points []Point, progress int, currentLon float64, currentLat float64) (RuntimeRouteMeta, error) {
	if len(points) == 0 {
		return RuntimeRouteMeta{}, fmt.Errorf("runtime route points empty")
	}

	routeVersion := newRouteVersion(vehicleID, shipmentID, sequential)
	clonedPoints := copyPoints(points)
	meta := buildRuntimeRouteMeta(vehicleID, shipmentID, sequential, routeVersion, clonedPoints, progress, currentLon, currentLat)

	cacheRoutePoints(routeVersion, clonedPoints)
	cacheRouteMeta(meta)

	if err := persistRouteToRedis(meta, clonedPoints); err != nil {
		Logger.Logger.Warn("写入 Redis 运行态路径失败，继续使用内存缓存", zap.Uint("vehicleId", vehicleID), zap.Error(err))
	}

	return meta, nil
}

func persistRouteToRedis(meta RuntimeRouteMeta, points []Point) error {
	if database.RedisClient == nil {
		return nil
	}

	pointsJSON, err := json.Marshal(points)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pipe := database.RedisClient.TxPipeline()
	pipe.Set(ctx, runtimeRoutePathKey(meta.RouteVersion), pointsJSON, runtimeRouteTTL())
	pipe.HSet(ctx, runtimeRouteVehicleKey(meta.VehicleID), map[string]interface{}{
		"vehicleId":    meta.VehicleID,
		"shipmentId":   meta.ShipmentID,
		"sequential":   meta.Sequential,
		"routeVersion": meta.RouteVersion,
		"progress":     meta.Progress,
		"total":        meta.Total,
		"currentLon":   meta.CurrentLon,
		"currentLat":   meta.CurrentLat,
		"updatedAt":    meta.UpdatedAt,
	})
	pipe.Expire(ctx, runtimeRouteVehicleKey(meta.VehicleID), runtimeRouteTTL())
	_, err = pipe.Exec(ctx)
	return err
}

func updateRuntimeRouteMeta(meta RuntimeRouteMeta) error {
	cacheRouteMeta(meta)
	if database.RedisClient == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pipe := database.RedisClient.TxPipeline()
	pipe.HSet(ctx, runtimeRouteVehicleKey(meta.VehicleID), map[string]interface{}{
		"vehicleId":    meta.VehicleID,
		"shipmentId":   meta.ShipmentID,
		"sequential":   meta.Sequential,
		"routeVersion": meta.RouteVersion,
		"progress":     meta.Progress,
		"total":        meta.Total,
		"currentLon":   meta.CurrentLon,
		"currentLat":   meta.CurrentLat,
		"updatedAt":    meta.UpdatedAt,
	})
	pipe.Expire(ctx, runtimeRouteVehicleKey(meta.VehicleID), runtimeRouteTTL())
	_, err := pipe.Exec(ctx)
	return err
}

func clearRuntimeRoute(vehicleID uint) {
	meta, _, ok := getRuntimeRouteState(vehicleID)
	if ok {
		dropRouteVersion(meta.RouteVersion)
	}
	dropVehicleMeta(vehicleID)

	if database.RedisClient == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pipe := database.RedisClient.TxPipeline()
	if ok && meta.RouteVersion != "" {
		pipe.Del(ctx, runtimeRoutePathKey(meta.RouteVersion))
	}
	pipe.Del(ctx, runtimeRouteVehicleKey(vehicleID))
	_, err := pipe.Exec(ctx)
	if err != nil && err != goRedis.Nil {
		Logger.Logger.Warn("清理 Redis 运行态路径失败", zap.Uint("vehicleId", vehicleID), zap.Error(err))
	}
}

func getRuntimeRouteState(vehicleID uint) (RuntimeRouteMeta, []Point, bool) {
	routeMetaCache.mu.RLock()
	meta, hasMeta := routeMetaCache.byVehicle[vehicleID]
	routeMetaCache.mu.RUnlock()
	if hasMeta {
		if points, ok := getRoutePointsByVersion(meta.RouteVersion); ok {
			return meta, points, true
		}
	}

	meta, ok := loadRuntimeRouteMetaFromRedis(vehicleID)
	if !ok {
		return RuntimeRouteMeta{}, nil, false
	}
	points, ok := loadRoutePointsFromRedis(meta.RouteVersion)
	if !ok {
		return RuntimeRouteMeta{}, nil, false
	}
	cacheRouteMeta(meta)
	cacheRoutePoints(meta.RouteVersion, points)
	return meta, points, true
}

func getRoutePointsByVersion(routeVersion string) ([]Point, bool) {
	routePathCache.mu.RLock()
	points, ok := routePathCache.byVersion[routeVersion]
	routePathCache.mu.RUnlock()
	if ok {
		return copyPoints(points), true
	}
	points, ok = loadRoutePointsFromRedis(routeVersion)
	if ok {
		cacheRoutePoints(routeVersion, points)
		return copyPoints(points), true
	}
	return nil, false
}

func loadRuntimeRouteMetaFromRedis(vehicleID uint) (RuntimeRouteMeta, bool) {
	if database.RedisClient == nil {
		return RuntimeRouteMeta{}, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	values, err := database.RedisClient.HGetAll(ctx, runtimeRouteVehicleKey(vehicleID)).Result()
	if err != nil || len(values) == 0 {
		return RuntimeRouteMeta{}, false
	}

	meta := RuntimeRouteMeta{
		VehicleID:    parseUintValue(values["vehicleId"]),
		ShipmentID:   parseUintValue(values["shipmentId"]),
		Sequential:   parseIntValue(values["sequential"]),
		RouteVersion: values["routeVersion"],
		Progress:     parseIntValue(values["progress"]),
		Total:        parseIntValue(values["total"]),
		CurrentLon:   parseFloatValue(values["currentLon"]),
		CurrentLat:   parseFloatValue(values["currentLat"]),
		UpdatedAt:    parseInt64Value(values["updatedAt"]),
	}
	if meta.VehicleID == 0 || meta.RouteVersion == "" {
		return RuntimeRouteMeta{}, false
	}
	return meta, true
}

func loadRoutePointsFromRedis(routeVersion string) ([]Point, bool) {
	if database.RedisClient == nil || routeVersion == "" {
		return nil, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	raw, err := database.RedisClient.Get(ctx, runtimeRoutePathKey(routeVersion)).Bytes()
	if err != nil {
		return nil, false
	}

	var points []Point
	if err = json.Unmarshal(raw, &points); err != nil || len(points) == 0 {
		return nil, false
	}
	return points, true
}

func parseUintValue(raw string) uint {
	value, _ := strconv.ParseUint(raw, 10, 64)
	return uint(value)
}

func parseIntValue(raw string) int {
	value, _ := strconv.Atoi(raw)
	return value
}

func parseInt64Value(raw string) int64 {
	value, _ := strconv.ParseInt(raw, 10, 64)
	return value
}

func parseFloatValue(raw string) float64 {
	value, _ := strconv.ParseFloat(raw, 64)
	return value
}

func findNearestProgress(points []Point, lon float64, lat float64) int {
	if len(points) == 0 {
		return 0
	}
	bestIdx := 0
	bestDistance := math.MaxFloat64
	for idx, point := range points {
		distance := haversineKM(lon, lat, point.Lon, point.Lat)
		if distance < bestDistance {
			bestDistance = distance
			bestIdx = idx
		}
	}
	return bestIdx
}

func alignRuntimeMetaToPosition(meta RuntimeRouteMeta, points []Point, lon float64, lat float64) RuntimeRouteMeta {
	if len(points) == 0 {
		return meta
	}
	currentIdx := clampProgress(meta.Progress, len(points))
	currentPoint := points[currentIdx]
	distance := haversineKM(lon, lat, currentPoint.Lon, currentPoint.Lat)
	if distance <= runtimeRouteAlignmentThresholdKM {
		meta.CurrentLon = lon
		meta.CurrentLat = lat
		meta.UpdatedAt = time.Now().UnixMilli()
		return meta
	}

	nearestIdx := findNearestProgress(points, lon, lat)
	meta.Progress = nearestIdx
	meta.CurrentLon = lon
	meta.CurrentLat = lat
	meta.UpdatedAt = time.Now().UnixMilli()
	return meta
}

func buildFullPathPayload(meta RuntimeRouteMeta, points []Point) map[string]interface{} {
	return map[string]interface{}{
		"vehicleId":    meta.VehicleID,
		"shipmentId":   meta.ShipmentID,
		"sequential":   meta.Sequential,
		"routeVersion": meta.RouteVersion,
		"progress":     meta.Progress,
		"total":        meta.Total,
		"points":       points,
	}
}

func buildProgressPayload(meta RuntimeRouteMeta, point Point) map[string]interface{} {
	return map[string]interface{}{
		"vehicleId":    meta.VehicleID,
		"shipmentId":   meta.ShipmentID,
		"sequential":   meta.Sequential,
		"routeVersion": meta.RouteVersion,
		"progress":     meta.Progress,
		"total":        meta.Total,
		"point": map[string]float64{
			"lon": point.Lon,
			"lat": point.Lat,
		},
	}
}

func buildClearPayload(meta RuntimeRouteMeta) map[string]interface{} {
	return map[string]interface{}{
		"vehicleId":    meta.VehicleID,
		"shipmentId":   meta.ShipmentID,
		"sequential":   meta.Sequential,
		"routeVersion": meta.RouteVersion,
	}
}
