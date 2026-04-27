package task

import (
	"fmt"
	"testing"
	"time"

	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/kiritosuki/mover/internal/config"
	"github.com/kiritosuki/mover/internal/constant"
	"github.com/kiritosuki/mover/internal/database"
	"github.com/kiritosuki/mover/internal/model"
	"github.com/kiritosuki/mover/internal/repository"
	"go.uber.org/zap"
)

// TestFullScheduleSimulation 完整调度模拟测试
// 1. 生成数据
// 2. 运行 CreateOrderTask 进行分配
// 3. 运行 SimulateMoving 推进
func TestFullScheduleSimulation(t *testing.T) {
	// 初始化
	Logger.InitLogger()
	config.InitViper()
	database.InitDB()
	db := database.DB

	// 清理并准备数据
	db.Exec("DELETE FROM order_task")
	db.Exec("DELETE FROM shipment")
	db.Exec("DELETE FROM vehicle")
	db.Exec("DELETE FROM cargo")
	db.Exec("DELETE FROM poi")

	// 1. POIs
	pois := []model.Poi{}
	for i := 1; i <= 5; i++ {
		p := model.Poi{Name: fmt.Sprintf("Poi-%d", i), Tybe: 1, Lon: 104.0 + float64(i)*0.01, Lat: 30.5 + float64(i)*0.01, Status: 1}
		db.Create(&p)
		pois = append(pois, p)
	}

	// 2. Cargos
	c1 := model.Cargo{Name: 1, Tybe: 1, Weight: 10} // 普货
	c2 := model.Cargo{Name: 2, Tybe: 2, Weight: 20} // 冷链
	db.Create(&c1)
	db.Create(&c2)

	// 3. Vehicles (10辆)
	for i := 1; i <= 10; i++ {
		vType := 1
		if i > 5 {
			vType = 2
		}
		v := model.Vehicle{
			Lon: 104.0, Lat: 30.5, Status: constant.VehicleStatusFree,
			Tybe: vType, Capacity: 1000, Size: 0, Speed: 1.0, UpdateTime: time.Now(),
		}
		db.Create(&v)
	}

	// 4. Shipments (15个)
	for i := 1; i <= 15; i++ {
		cId := c1.Id
		if i > 8 {
			cId = c2.Id
		}
		s := model.Shipment{
			StartPoiId: pois[0].Id, EndPoiId: pois[i%5].Id,
			CargoId: cId, Count: 5, Status: constant.ShipmentStatusSleeping,
			CreateTime: time.Now(), UpdateTime: time.Now(),
		}
		db.Create(&s)
	}

	Logger.Logger.Info("测试数据准备就绪，开始模拟调度...")

	// 运行一次调度分配
	shipments, _ := repository.GetSleepingShipment()
	if len(shipments) > 0 {
		Logger.Logger.Info("执行一批次调度分配", zap.Int("count", len(shipments)))
		doCreateOrderTaskBatch(shipments)
	}

	// 检查分配结果
	var taskCount int64
	db.Model(&model.OrderTask{}).Count(&taskCount)
	Logger.Logger.Info("任务分配完成", zap.Int64("tasksCreated", taskCount))

	if taskCount == 0 {
		t.Fatal("未创建任何任务，调度算法可能存在问题")
	}

	// 运行一两次移动模拟
	tasks, _ := repository.GetRunningTasksOrderedByVehicle()
	Logger.Logger.Info("推进车辆移动", zap.Int("runningTasks", len(tasks)))
	
	// 模拟几次移动循环
	for i := 0; i < 3; i++ {
		movingVehicles := make(map[uint]bool)
		for _, task := range tasks {
			if movingVehicles[task.VehicleId] {
				continue
			}
			movingVehicles[task.VehicleId] = true
			moveVehicle(task) // 同步运行以便观察
		}
		time.Sleep(100 * time.Millisecond)
	}

	Logger.Logger.Info("模拟测试结束")
}
