package task

import (
	"testing"
	"time"

	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/kiritosuki/mover/internal/config"
	"github.com/kiritosuki/mover/internal/constant"
	"github.com/kiritosuki/mover/internal/database"
	"github.com/kiritosuki/mover/internal/model"
	"github.com/kiritosuki/mover/internal/repository"
)

// TestLevelFactor 验证司机等级对概率的修正系数边界。
func TestLevelFactor(t *testing.T) {
	old := randomEventCfg
	defer func() { randomEventCfg = old }()
	randomEventCfg.LevelFactor = [5]float64{1.4, 1.2, 1.0, 0.85, 0.7}

	if got := levelFactor(1); got != 1.4 {
		t.Fatalf("level 1 expected 1.4, got %v", got)
	}
	if got := levelFactor(5); got != 0.7 {
		t.Fatalf("level 5 expected 0.7, got %v", got)
	}
	if got := levelFactor(0); got != 1.4 {
		t.Fatalf("level 0 clamps to 1, expected 1.4, got %v", got)
	}
	if got := levelFactor(9); got != 0.7 {
		t.Fatalf("level 9 clamps to 5, expected 0.7, got %v", got)
	}
}

// TestRandomRangeSeconds 验证随机时长区间取值落在边界内。
func TestRandomRangeSeconds(t *testing.T) {
	for i := 0; i < 100; i++ {
		v := randomRangeSeconds([2]float64{60, 120})
		if v < 60 || v > 120 {
			t.Fatalf("randomRangeSeconds out of range: %d", v)
		}
	}
	// 等边界时恒等
	if v := randomRangeSeconds([2]float64{90, 90}); v != 90 {
		t.Fatalf("equal bounds should return bound, got %d", v)
	}
}

// TestIsVehiclePaused 验证暂停判定逻辑。
func TestIsVehiclePaused(t *testing.T) {
	if isVehiclePaused(nil) {
		t.Fatal("nil vehicle should not be paused")
	}
	if isVehiclePaused(&model.Vehicle{}) {
		t.Fatal("vehicle without pause_until should not be paused")
	}

	future := time.Now().Add(10 * time.Minute)
	if !isVehiclePaused(&model.Vehicle{PauseUntil: &future}) {
		t.Fatal("vehicle with future pause_until should be paused")
	}

	past := time.Now().Add(-10 * time.Minute)
	if isVehiclePaused(&model.Vehicle{PauseUntil: &past}) {
		t.Fatal("vehicle with past pause_until should NOT be paused")
	}
}

// TestRefuseAssignmentFlow 真实跑库集成测试：司机拒单回退闭环。
// 1) 创建车辆+司机+运单+Planned 任务；
// 2) 强制拒单概率=1，验证任务删除、运单回退 Sleeping、预占载重释放、车辆 Free。
func TestRefuseAssignmentFlow(t *testing.T) {
	Logger.InitLogger()
	config.InitViper()
	database.InitDB()
	db := database.DB

	db.Exec("DELETE FROM random_event")
	db.Exec("DELETE FROM driver")
	db.Exec("DELETE FROM order_task")
	db.Exec("DELETE FROM shipment")
	db.Exec("DELETE FROM cargo")
	db.Exec("DELETE FROM vehicle")

	cargo := model.Cargo{Name: "普货", Tybe: 1, Weight: 100}
	if err := db.Create(&cargo).Error; err != nil {
		t.Fatalf("create cargo failed: %v", err)
	}
	vehicle := model.Vehicle{License: "测试-拒单", Status: constant.VehicleStatusReady, Tybe: 1, Capacity: 1000, ReservedSize: 200, UpdateTime: time.Now()}
	if err := db.Create(&vehicle).Error; err != nil {
		t.Fatalf("create vehicle failed: %v", err)
	}
	driver := model.Driver{Name: "测试司机", Level: 1, Status: 1, VehicleId: vehicle.Id, UpdateTime: time.Now()}
	if err := db.Create(&driver).Error; err != nil {
		t.Fatalf("create driver failed: %v", err)
	}
	shipment := model.Shipment{StartPoiId: 1, EndPoiId: 2, CargoId: cargo.Id, Count: 2, Status: constant.ShipmentStatusQueued, CreateTime: time.Now(), UpdateTime: time.Now()}
	if err := db.Create(&shipment).Error; err != nil {
		t.Fatalf("create shipment failed: %v", err)
	}
	now := time.Now()
	task := model.OrderTask{ShipmentId: shipment.Id, VehicleId: vehicle.Id, Sequential: constant.OrderTaskSequentialPlanned, CreateTime: now, UpdateTime: now}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create task failed: %v", err)
	}

	// 强制拒单
	old := randomEventCfg
	defer func() { randomEventCfg = old }()
	randomEventCfg.Enabled = true
	randomEventCfg.DriverRefuseProb = 1.0
	randomEventCfg.RefuseMaxPerShipment = 2

	reloadedVehicle, _ := repository.GetVehicle(int(vehicle.Id))
	reloadedShipment, _ := repository.GetShipment(int(shipment.Id))
	if !maybeRefuseAssignment(&task, reloadedVehicle, reloadedShipment) {
		t.Fatal("expected refuse to trigger")
	}

	// 验证任务已删除
	var taskCount int64
	db.Model(&model.OrderTask{}).Where("id = ?", task.Id).Count(&taskCount)
	if taskCount != 0 {
		t.Fatalf("task should be deleted, count=%d", taskCount)
	}

	// 验证运单回退 Sleeping
	updatedShipment, _ := repository.GetShipment(int(shipment.Id))
	if updatedShipment.Status != constant.ShipmentStatusSleeping {
		t.Fatalf("shipment should be Sleeping, got %d", updatedShipment.Status)
	}

	// 验证预占载重释放 (200-200=0)
	updatedVehicle, _ := repository.GetVehicle(int(vehicle.Id))
	if updatedVehicle.ReservedSize != 0 {
		t.Fatalf("reserved size should be released to 0, got %d", updatedVehicle.ReservedSize)
	}

	// 验证车辆回退空闲
	if updatedVehicle.Status != constant.VehicleStatusFree {
		t.Fatalf("vehicle should be Free after refuse, got %d", updatedVehicle.Status)
	}

	// 验证拒单事件落库
	refuseCount, _ := repository.CountRefuseByShipment(shipment.Id)
	if refuseCount != 1 {
		t.Fatalf("refuse event count should be 1, got %d", refuseCount)
	}
}

// TestAccidentAndRecoveryFlow 真实跑库集成测试：车祸→维修→恢复闭环。
// 1) 车辆 Running + 在途任务；
// 2) 强制车祸概率=1 触发 → 车辆转维修中 + pause_until 设置；
// 3) 维修到期后 recoverMaintenanceVehicles → 恢复 Running 续跑在途任务。
func TestAccidentAndRecoveryFlow(t *testing.T) {
	Logger.InitLogger()
	config.InitViper()
	database.InitDB()
	db := database.DB

	db.Exec("DELETE FROM random_event")
	db.Exec("DELETE FROM driver")
	db.Exec("DELETE FROM order_task")
	db.Exec("DELETE FROM shipment")
	db.Exec("DELETE FROM cargo")
	db.Exec("DELETE FROM vehicle")

	cargo := model.Cargo{Name: "冷链", Tybe: 2, Weight: 100}
	if err := db.Create(&cargo).Error; err != nil {
		t.Fatalf("create cargo failed: %v", err)
	}
	vehicle := model.Vehicle{License: "测试-车祸", Status: constant.VehicleStatusRunning, Tybe: 2, Capacity: 1000, UpdateTime: time.Now()}
	if err := db.Create(&vehicle).Error; err != nil {
		t.Fatalf("create vehicle failed: %v", err)
	}
	driver := model.Driver{Name: "车祸司机", Level: 3, Status: 1, VehicleId: vehicle.Id, UpdateTime: time.Now()}
	if err := db.Create(&driver).Error; err != nil {
		t.Fatalf("create driver failed: %v", err)
	}
	shipment := model.Shipment{StartPoiId: 1, EndPoiId: 2, CargoId: cargo.Id, Count: 1, Status: constant.ShipmentStatusWorking, CreateTime: time.Now(), UpdateTime: time.Now()}
	if err := db.Create(&shipment).Error; err != nil {
		t.Fatalf("create shipment failed: %v", err)
	}
	now := time.Now()
	task := model.OrderTask{ShipmentId: shipment.Id, VehicleId: vehicle.Id, Sequential: constant.OrderTaskSequentialTransporting, CreateTime: now, UpdateTime: now}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create task failed: %v", err)
	}

	// 强制车祸
	old := randomEventCfg
	defer func() { randomEventCfg = old }()
	randomEventCfg.Enabled = true
	randomEventCfg.AccidentProbPerTick = 1.0
	randomEventCfg.MaintenanceSeconds = [2]float64{60, 60}

	reloadedVehicle, _ := repository.GetVehicle(int(vehicle.Id))
	if !tryTriggerTransportRandomEvents(&task, reloadedVehicle) {
		t.Fatal("expected accident to trigger")
	}

	// 验证车辆转维修中
	updatedVehicle, _ := repository.GetVehicle(int(vehicle.Id))
	if updatedVehicle.Status != constant.VehicleStatusMaintenance {
		t.Fatalf("vehicle should be Maintenance, got %d", updatedVehicle.Status)
	}
	if updatedVehicle.PauseUntil == nil {
		t.Fatal("pause_until should be set after accident")
	}

	// 将 pause_until 改为过去时间，模拟维修到期
	past := time.Now().Add(-1 * time.Minute)
	_ = repository.UpdateVehicleFields(int(vehicle.Id), map[string]interface{}{"pause_until": past})

	// 运行恢复逻辑
	recoverMaintenanceVehicles()

	// 车辆应恢复 Running(因为有在途任务)
	recovered, _ := repository.GetVehicle(int(vehicle.Id))
	if recovered.Status != constant.VehicleStatusRunning {
		t.Fatalf("vehicle should be Running after recovery, got %d", recovered.Status)
	}

	// 司机应复位可用
	recoveredDriver, _ := repository.GetDriverByVehicle(vehicle.Id)
	if recoveredDriver.Status != 1 {
		t.Fatalf("driver should be available after recovery, got %d", recoveredDriver.Status)
	}
}

// TestRestTrigger 真实跑库集成测试：疲劳累计达到阈值触发强制休息。
func TestRestTrigger(t *testing.T) {
	Logger.InitLogger()
	config.InitViper()
	database.InitDB()
	db := database.DB

	db.Exec("DELETE FROM random_event")
	db.Exec("DELETE FROM driver")
	db.Exec("DELETE FROM order_task")
	db.Exec("DELETE FROM shipment")
	db.Exec("DELETE FROM cargo")
	db.Exec("DELETE FROM vehicle")

	vehicle := model.Vehicle{License: "测试-休息", Status: constant.VehicleStatusRunning, Tybe: 1, Capacity: 1000, UpdateTime: time.Now()}
	if err := db.Create(&vehicle).Error; err != nil {
		t.Fatalf("create vehicle failed: %v", err)
	}
	driver := model.Driver{Name: "休息司机", Level: 2, Status: 1, VehicleId: vehicle.Id, UpdateTime: time.Now()}
	if err := db.Create(&driver).Error; err != nil {
		t.Fatalf("create driver failed: %v", err)
	}

	// 强制休息阈值极低
	old := randomEventCfg
	defer func() { randomEventCfg = old }()
	randomEventCfg.Enabled = true
	randomEventCfg.RestAfterHours = 0.0001
	randomEventCfg.RestMinutes = 20

	task := model.OrderTask{ShipmentId: 999, VehicleId: vehicle.Id, Sequential: constant.OrderTaskSequentialTransporting}
	accumulateDriverFatigue(&task, &vehicle, 0.1)

	updatedDriver, _ := repository.GetDriverByVehicle(vehicle.Id)
	if updatedDriver.Status != 2 {
		t.Fatalf("driver should be resting, got %d", updatedDriver.Status)
	}
	if updatedDriver.RestUntil == nil {
		t.Fatal("rest_until should be set")
	}

	// 车辆同步暂停
	updatedVehicle, _ := repository.GetVehicle(int(vehicle.Id))
	if updatedVehicle.PauseUntil == nil {
		t.Fatal("vehicle pause_until should be set during rest")
	}

	// 事件落库
	var restCount int64
	db.Model(&model.RandomEvent{}).Where("event_type = ?", "rest").Count(&restCount)
	if restCount != 1 {
		t.Fatalf("rest event should be recorded, got %d", restCount)
	}
}