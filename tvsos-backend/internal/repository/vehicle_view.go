package repository

import (
	"strconv"
	"time"

	"github.com/kiritosuki/mover/internal/constant"
	"github.com/kiritosuki/mover/internal/model"
)

const defaultVehicleAngle = 90.0

type VehicleView struct {
	Id            uint      `json:"id"`
	License       string    `json:"license"`
	Status        int       `json:"status"`
	Lon           float64   `json:"lon"`
	Lat           float64   `json:"lat"`
	Speed         float64   `json:"speed"`
	UpdateTime    time.Time `json:"updateTime"`
	CategoryId    int       `json:"categoryId"`
	Tybe          int       `json:"tybe"`
	Size          int       `json:"size"`
	ReservedSize  int       `json:"reservedSize"`
	Capacity      int       `json:"capacity"`
	Length        float64   `json:"length"`
	Width         float64   `json:"width"`
	Height        float64   `json:"height"`
	WaitTime      float64   `json:"waitTime"`
	TotalWaitTime float64   `json:"totalWaitTime"`
	EmptyMileage  float64   `json:"emptyMileage"`
	Distance      float64   `json:"distance"`
	Duration      float64   `json:"duration"`
	Angle         float64   `json:"angle"`
}

// BuildVehicleView 用于将底层的 model.Vehicle 模型转换为前端可直接展示的 DTO 对象 (VehicleView)。
// 同时对一些空字段(缺省车牌License和朝向Angle)赋予默认值，
// 并计算如果处于空闲状态，依据 update_time 动态叠加并推演实时的 WaitTime。
func BuildVehicleView(v *model.Vehicle) *VehicleView {
	if v == nil {
		return nil
	}

	license := v.License
	if license == "" {
		license = "车辆-" + strconv.Itoa(int(v.Id))
	}

	angle := v.Angle
	if angle == 0 {
		angle = defaultVehicleAngle
	}

	waitTime := v.WaitTime
	totalWaitTime := v.TotalWaitTime
	if v.Status == constant.VehicleStatusFree && !v.UpdateTime.IsZero() {
		liveWait := time.Since(v.UpdateTime).Minutes()
		if liveWait > 0 {
			waitTime += liveWait
			totalWaitTime += liveWait
		}
	}

	return &VehicleView{
		Id:            v.Id,
		License:       license,
		Status:        v.Status,
		Lon:           v.Lon,
		Lat:           v.Lat,
		Speed:         v.Speed,
		UpdateTime:    v.UpdateTime,
		CategoryId:    v.Tybe,
		Tybe:          v.Tybe,
		Size:          v.Size,
		ReservedSize:  v.ReservedSize,
		Capacity:      v.Capacity,
		Length:        v.Length,
		Width:         v.Width,
		Height:        v.Height,
		WaitTime:      waitTime,
		TotalWaitTime: totalWaitTime,
		EmptyMileage:  v.EmptyMileage,
		Distance:      v.Distance,
		Duration:      v.Duration,
		Angle:         angle,
	}
}

// BuildVehicleViews 是一个辅助生成函数。
// 用于遍历组装切片，将多个 model.Vehicle 转换为对应的 VehicleView 切片集合并返回。
func BuildVehicleViews(vehicles []*model.Vehicle) []*VehicleView {
	views := make([]*VehicleView, 0, len(vehicles))
	for _, v := range vehicles {
		views = append(views, BuildVehicleView(v))
	}
	return views
}
