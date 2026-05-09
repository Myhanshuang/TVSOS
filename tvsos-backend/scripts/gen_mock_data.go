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
	// 先将货物表结构进行变更防御 (将 id 和 name 类型改成支持我们新的字符串和扩充需求)
	db.Exec("ALTER TABLE cargo MODIFY COLUMN name VARCHAR(255);")

	// 清理旧数据 (因为我们要重新配置固化的货物和车辆)
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.Shipment{})
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.OrderTask{}) // 关联的外键也要清理
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.Vehicle{})
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.Cargo{})

	// 1. 获取现有真实 POI 点作为发货地和目的地
	var pois []model.Poi
	db.Find(&pois)
	if len(pois) < 2 {
		fmt.Println("警告：数据库中没有足够的 POI 数据，请先运行 import_poi.go 脚本导入真实的 poi.json！")
		return
	}

	// 2. 生成固化的货物种类 (10种)
	// 根据不同货物类型与推荐运输车辆进行简单的关联说明：
	cargos := []model.Cargo{
		{Id: 1, Name: "普通包裹", Tybe: 1, Pack: 1, Weight: 10},      // 适配厢货/面包车
		{Id: 2, Name: "生鲜食品", Tybe: 2, Pack: 2, Weight: 50},      // 适配冷藏车
		{Id: 3, Name: "大型机械", Tybe: 3, Pack: 3, Weight: 2000},    // 适配平板车/高栏车
		{Id: 4, Name: "建材石料", Tybe: 4, Pack: 3, Weight: 5000},    // 适配平板车
		{Id: 5, Name: "危险化学品", Tybe: 5, Pack: 4, Weight: 1000},   // 适配危化品罐车
		{Id: 6, Name: "农副产品", Tybe: 6, Pack: 1, Weight: 500},     // 适配高栏车/厢货
		{Id: 7, Name: "医药及疫苗", Tybe: 7, Pack: 2, Weight: 5},      // 适配冷藏车
		{Id: 8, Name: "家用电器", Tybe: 8, Pack: 1, Weight: 100},     // 适配厢货
		{Id: 9, Name: "汽车配件", Tybe: 9, Pack: 1, Weight: 200},     // 适配厢货/高栏车
		{Id: 10, Name: "生猪活禽", Tybe: 10, Pack: 5, Weight: 300},   // 适配高栏车
	}
	for i := range cargos {
		db.Create(&cargos[i])
	}

	// 3. 生成固化的 20 辆货车 (每种类型一辆或多辆)
	vehicles := []model.Vehicle{
		{Id: 1, License: "川A·NORMAL1", Tybe: 1, Capacity: 5000, Length: 6.8, Width: 2.4, Height: 2.6, Speed: 50, Status: 2}, // 普通厢式货车
		{Id: 2, License: "川A·COLD001", Tybe: 2, Capacity: 3000, Length: 5.2, Width: 2.2, Height: 2.2, Speed: 60, Status: 2}, // 冷藏车
		{Id: 3, License: "川A·FLATB01", Tybe: 3, Capacity: 15000, Length: 13.0, Width: 2.5, Height: 1.5, Speed: 40, Status: 2}, // 平板车
		{Id: 4, License: "川A·TANKER1", Tybe: 4, Capacity: 20000, Length: 10.0, Width: 2.5, Height: 3.5, Speed: 40, Status: 2}, // 危化品罐车
		{Id: 5, License: "川A·HIGH001", Tybe: 5, Capacity: 10000, Length: 9.6, Width: 2.4, Height: 3.0, Speed: 45, Status: 2},  // 高栏车
		{Id: 6, License: "川A·MINI001", Tybe: 6, Capacity: 800, Length: 2.8, Width: 1.5, Height: 1.3, Speed: 70, Status: 2},   // 微型面包车
		// 增加至 20 辆
		{Id: 7, License: "川A·NORMAL2", Tybe: 1, Capacity: 5000, Length: 6.8, Width: 2.4, Height: 2.6, Speed: 50, Status: 2},
		{Id: 8, License: "川A·NORMAL3", Tybe: 1, Capacity: 8000, Length: 7.6, Width: 2.4, Height: 2.6, Speed: 50, Status: 2},
		{Id: 9, License: "川A·NORMAL4", Tybe: 1, Capacity: 8000, Length: 7.6, Width: 2.4, Height: 2.6, Speed: 50, Status: 2},
		{Id: 10, License: "川A·NORMAL5", Tybe: 1, Capacity: 5000, Length: 6.8, Width: 2.4, Height: 2.6, Speed: 50, Status: 2},
		{Id: 11, License: "川A·COLD002", Tybe: 2, Capacity: 3000, Length: 5.2, Width: 2.2, Height: 2.2, Speed: 60, Status: 2},
		{Id: 12, License: "川A·COLD003", Tybe: 2, Capacity: 5000, Length: 6.8, Width: 2.4, Height: 2.4, Speed: 60, Status: 2},
		{Id: 13, License: "川A·FLATB02", Tybe: 3, Capacity: 15000, Length: 13.0, Width: 2.5, Height: 1.5, Speed: 40, Status: 2},
		{Id: 14, License: "川A·FLATB03", Tybe: 3, Capacity: 15000, Length: 13.0, Width: 2.5, Height: 1.5, Speed: 40, Status: 2},
		{Id: 15, License: "川A·TANKER2", Tybe: 4, Capacity: 20000, Length: 10.0, Width: 2.5, Height: 3.5, Speed: 40, Status: 2},
		{Id: 16, License: "川A·HIGH002", Tybe: 5, Capacity: 10000, Length: 9.6, Width: 2.4, Height: 3.0, Speed: 45, Status: 2},
		{Id: 17, License: "川A·HIGH003", Tybe: 5, Capacity: 10000, Length: 9.6, Width: 2.4, Height: 3.0, Speed: 45, Status: 2},
		{Id: 18, License: "川A·MINI002", Tybe: 6, Capacity: 800, Length: 2.8, Width: 1.5, Height: 1.3, Speed: 70, Status: 2},
		{Id: 19, License: "川A·MINI003", Tybe: 6, Capacity: 800, Length: 2.8, Width: 1.5, Height: 1.3, Speed: 70, Status: 2},
		{Id: 20, License: "川A·MINI004", Tybe: 6, Capacity: 1000, Length: 3.2, Width: 1.6, Height: 1.4, Speed: 70, Status: 2},
	}
	
	for i := range vehicles {
		v := &vehicles[i]
		// 随机赋予初始位置（直接从真实 POI 中选一个作为初始发车地）
		v.Lon = pois[rand.Intn(len(pois))].Lon
		v.Lat = pois[rand.Intn(len(pois))].Lat
		v.UpdateTime = time.Now()
		v.WaitTime = 0
		v.TotalWaitTime = 0
		v.EmptyMileage = 0

		db.Create(v)
	}

	// 4. 生成一批随机货物订单 (Shipments)
	rand.Seed(time.Now().UnixNano())
	
	// 订单总数定义为 50
	totalShipments := 50
	for i := 0; i < totalShipments; i++ {
		// 为了保证每种货物（比如危险化学品）必定出现在图表中，前 10 票我们强制按顺序遍历各种货物，后 40 票随机
		var cargo model.Cargo
		if i < len(cargos) {
			cargo = cargos[i]
		} else {
			cargo = cargos[rand.Intn(len(cargos))]
		}

		startPoi := pois[rand.Intn(len(pois))]
		endPoi := pois[rand.Intn(len(pois))]
		for startPoi.Id == endPoi.Id {
			endPoi = pois[rand.Intn(len(pois))]
		}

		// 核心修改点：为了拉高系统的总载重使用率（约80%），需要把总目标重量调大。
		// 目前总运能大约在 17万 kg (20辆车加起来)，50个单子要想占到 13.6万 kg，也就是每单平均需要 2700kg-3000kg 左右才行
		// 但是考虑到还有分配率和一些车装不满，我们将票货的总目标重量放大范围为 4000kg 到 12000kg 左右，让车辆可以满载
		targetWeight := 4000 + rand.Intn(8000)
		count := targetWeight / cargo.Weight
		if count <= 0 {
			count = 1
		}

		s := model.Shipment{
			StartPoiId: startPoi.Id,
			EndPoiId:   endPoi.Id,
			CargoId:    cargo.Id,
			Count:      count,             // 基于目标分布重量算出的箱数
			Status:     1,                 // Sleeping (待分配)
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		}
		db.Create(&s)
	}

	fmt.Printf("模拟数据生成完毕：基于 %d 个真实 POIs, 固化了 10 种 Cargos 和 20 辆真实映射车(Vehicles), 并新生成了 %d 笔带有合理约束因子的订单(Shipments)\n", len(pois), totalShipments)
}
