package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/kiritosuki/mover/internal/config"
	"github.com/kiritosuki/mover/internal/database"
	"github.com/kiritosuki/mover/internal/model"
	"gorm.io/gorm"
)

func main() {
	// 初始化配置和数据库
	Logger.InitLogger()
	config.InitViper()
	database.InitDB()
	db := database.DB

	// 清理旧数据 (可选，为了测试纯净性)
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.OrderTask{})
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.Shipment{})
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.Vehicle{})
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.Cargo{})
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.Poi{})

	// 1. 生成 POI 点 (10张，包含仓库和目的地)
	pois := []model.Poi{}
	for i := 1; i <= 20; i++ {
		p := model.Poi{
			Name:   fmt.Sprintf("POI-%d", i),
			Tybe:   rand.Intn(3) + 1,
			Lon:    104.0 + rand.Float64()*0.1,
			Lat:    30.5 + rand.Float64()*0.1,
			Status: 1,
		}
		db.Create(&p)
		pois = append(pois, p)
	}

	// 2. 生成货物种类 (1-普通, 2-冷链)
	cargos := []model.Cargo{
		{Name: 101, Tybe: 1, Pack: 1, Weight: 10}, // 普通货 A (10kg/箱)
		{Name: 102, Tybe: 2, Pack: 1, Weight: 20}, // 冷链货 B (20kg/箱)
	}
	for i := range cargos {
		db.Create(&cargos[i])
	}

	// 3. 生成 10 辆货车 (5辆普通, 5辆冷链)
	for i := 1; i <= 10; i++ {
		vType := 1
		if i > 5 {
			vType = 2
		}
		v := model.Vehicle{
			License:       fmt.Sprintf("川A%05d", i),
			Lon:           104.05 + rand.Float64()*0.02,
			Lat:           30.55 + rand.Float64()*0.02,
			Status:        2, // Free
			Tybe:          vType,
			Capacity:      1000, // 1000kg
			Size:          0,
			ReservedSize:  0,
			Speed:         35,
			Length:        6.2 + rand.Float64()*1.2,
			Width:         2.2 + rand.Float64()*0.3,
			Height:        2.4 + rand.Float64()*0.4,
			WaitTime:      rand.Float64() * 20,
			TotalWaitTime: 100 + rand.Float64()*300,
			EmptyMileage:  rand.Float64() * 50,
			Distance:      rand.Float64() * 100,
			Duration:      rand.Float64() * 5,
			Angle:         90,
			UpdateTime:    time.Now(),
		}
		db.Create(&v)
	}

	// 4. 生成一批随机货物订单 (Shipments)
	rand.Seed(time.Now().UnixNano())
	for i := 1; i <= 30; i++ {
		cargo := cargos[rand.Intn(len(cargos))]
		startPoi := pois[rand.Intn(len(pois))]
		endPoi := pois[rand.Intn(len(pois))]
		if startPoi.Id == endPoi.Id {
			endPoi = pois[(rand.Intn(len(pois))+1)%len(pois)]
		}

		s := model.Shipment{
			StartPoiId: startPoi.Id,
			EndPoiId:   endPoi.Id,
			CargoId:    cargo.Id,
			Count:      rand.Intn(10) + 1, // 1-10箱
			Status:     1,                 // Sleeping
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		}
		db.Create(&s)
	}

	fmt.Println("模拟数据生成完毕：20 POIs, 2 Cargos, 10 Vehicles, 30 Shipments")
}
