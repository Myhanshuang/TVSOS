package handler

import (
	"errors"
	"math/rand"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kiritosuki/mover/internal/constant"
	"github.com/kiritosuki/mover/internal/model"
	"github.com/kiritosuki/mover/internal/repository"
	"github.com/kiritosuki/mover/internal/result"
)

type shipmentListItem struct {
	Num          uint      `json:"num"`
	BeginLon     float64   `json:"beginLon"`
	BeginLat     float64   `json:"beginLat"`
	EndLon       float64   `json:"endLon"`
	EndLat       float64   `json:"endLat"`
	EstBeginTime time.Time `json:"estBeginTime"`
	EstEndTime   time.Time `json:"estEndTime"`
	CreateTime   time.Time `json:"createTime"`
	Status       int       `json:"status"`
}

// ListShipments 查询订单列表，兼容前端 taskManage 字段
func ListShipments(c *gin.Context) {
	shipments, err := repository.ListShipments()
	if err != nil {
		result.Fail(c, "查询订单失败", err)
		return
	}

	numQuery := c.Query("num")
	statusQuery := c.Query("status")
	statusFilter := -1
	if statusQuery != "" {
		if v, parseErr := strconv.Atoi(statusQuery); parseErr == nil {
			statusFilter = v
		}
	}

	items := make([]shipmentListItem, 0, len(shipments))
	for _, shipment := range shipments {
		if numQuery != "" {
			if strconv.FormatUint(uint64(shipment.Id), 10) != numQuery {
				continue
			}
		}
		if statusFilter >= 0 && shipment.Status != statusFilter {
			continue
		}

		startPoi, startErr := repository.GetPoi(int(shipment.StartPoiId))
		if startErr != nil {
			continue
		}
		endPoi, endErr := repository.GetPoi(int(shipment.EndPoiId))
		if endErr != nil {
			continue
		}

		estBegin := shipment.UpdateTime
		if estBegin.IsZero() {
			estBegin = shipment.CreateTime
		}
		estEnd := estBegin.Add(30 * time.Minute)

		items = append(items, shipmentListItem{
			Num:          shipment.Id,
			BeginLon:     startPoi.Lon,
			BeginLat:     startPoi.Lat,
			EndLon:       endPoi.Lon,
			EndLat:       endPoi.Lat,
			EstBeginTime: estBegin,
			EstEndTime:   estEnd,
			CreateTime:   shipment.CreateTime,
			Status:       shipment.Status,
		})
	}

	result.Success(c, items)
}

// MockShipments 生成模拟订单
func MockShipments(c *gin.Context) {
	count, err := strconv.Atoi(c.Param("count"))
	if err != nil || count <= 0 {
		result.Fail(c, "count 必须是正整数", errors.New("invalid count"))
		return
	}

	pois, err := repository.ListAllPois()
	if err != nil {
		result.Fail(c, "查询POI失败", err)
		return
	}
	if len(pois) < 2 {
		result.Fail(c, "POI数量不足，至少需要2个", errors.New("not enough pois"))
		return
	}

	cargos, err := repository.ListCargos()
	if err != nil {
		result.Fail(c, "查询货物类型失败", err)
		return
	}
	if len(cargos) == 0 {
		result.Fail(c, "货物类型为空", errors.New("cargo list is empty"))
		return
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	created := 0
	for i := 0; i < count; i++ {
		startIndex := rng.Intn(len(pois))
		endIndex := rng.Intn(len(pois) - 1)
		if endIndex >= startIndex {
			endIndex++
		}

		cargo := cargos[rng.Intn(len(cargos))]
		shipment := &model.Shipment{
			StartPoiId: uint(pois[startIndex].Id),
			EndPoiId:   uint(pois[endIndex].Id),
			Status:     constant.ShipmentStatusSleeping,
			CargoId:    cargo.Id,
			Count:      rng.Intn(5) + 1,
		}
		if createErr := repository.CreateShipment(shipment); createErr == nil {
			created++
		}
	}

	result.Success(c, gin.H{"created": created})
}
