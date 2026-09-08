package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kiritosuki/mover/internal/repository"
	"github.com/kiritosuki/mover/internal/result"
	"github.com/kiritosuki/mover/internal/task"
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

// ListShipments 处理客户端查询运单(Shipment)列表的请求。
// 兼容前端 taskManage 组件的搜索字段，支持用 ID (num) 和 状态(status) 进行筛选过滤。
// 自动组装起点(StartPoi)和终点(EndPoi)的经纬度数据并返回。
func ListShipments(c *gin.Context) {
	// 沙箱系统：优先活跃实验运单；无运行中实验时回落模板/全局(experiment_id=0)；支持显式 experiment_id。
	expID := task.CurrentExperimentID()
	if expStr := c.Query("experiment_id"); expStr != "" {
		if v, parseErr := strconv.Atoi(expStr); parseErr == nil && v >= 0 {
			expID = uint(v)
		}
	}
	shipments, err := repository.ListShipmentsByExp(expID)
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
