package task

import (
	"math/rand"
	"time"

	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/kiritosuki/mover/internal/config"
	"github.com/kiritosuki/mover/internal/constant"
	"github.com/kiritosuki/mover/internal/model"
	"github.com/kiritosuki/mover/internal/repository"
	"go.uber.org/zap"
)

// RandomEventConfig 随机事件概率与参数配置（对应 config-dev.yml 的 random_events 段）
type RandomEventConfig struct {
	Enabled               bool
	DriverRefuseProb      float64   // 单次派单拒单基准概率
	RefuseMaxPerShipment  int       // 同一运单最多被拒次数(防死循环)
	RestAfterHours        float64   // 连续驾驶阈值(小时)
	RestMinutes           float64   // 单次休息时长(分钟)
	WeatherProbPerTick    float64   // 每 tick 天气触发概率
	WeatherDelaySeconds   [2]float64 // 天气延后区间(秒)
	AccidentProbPerTick   float64   // 每 tick 车祸触发概率
	MaintenanceSeconds    [2]float64 // 维修时长区间(秒)
	LevelFactor           [5]float64 // 司机等级 1-5 对拒单/事故概率的修正系数
}

var randomEventCfg = defaultRandomEventConfig()

func defaultRandomEventConfig() RandomEventConfig {
	return RandomEventConfig{
		Enabled:              true,
		DriverRefuseProb:     0.06,
		RefuseMaxPerShipment: 2,
		RestAfterHours:       4.0,
		RestMinutes:          20.0,
		WeatherProbPerTick:   0.0005,
		WeatherDelaySeconds:  [2]float64{1800, 7200},
		AccidentProbPerTick:  0.0002,
		MaintenanceSeconds:   [2]float64{3600, 21600},
		LevelFactor:          [5]float64{1.4, 1.2, 1.0, 0.85, 0.7},
	}
}

// getFloat64SliceCfg 从 viper 读取 float64 切片配置（兼容 Get 返回 []interface{} 的情况）。
func getFloat64SliceCfg(key string) []float64 {
	if config.VP == nil {
		return nil
	}
	raw := config.VP.Get(key)
	switch v := raw.(type) {
	case []interface{}:
		out := make([]float64, 0, len(v))
		for _, item := range v {
			switch num := item.(type) {
			case float64:
				out = append(out, num)
			case int:
				out = append(out, float64(num))
			}
		}
		return out
	case []float64:
		return v
	}
	return nil
}
// LoadRandomEventConfig 从 viper 配置读取 random_events 段，读取失败时保持默认值。
func LoadRandomEventConfig() {
	if config.VP == nil {
		return
	}
	cfg := randomEventCfg
	cfg.Enabled = config.VP.GetBool("random_events.enabled")
	if p := config.VP.GetFloat64("random_events.driver_refuse_prob"); p > 0 {
		cfg.DriverRefuseProb = p
	}
	if n := config.VP.GetInt("random_events.refuse_max_per_shipment"); n > 0 {
		cfg.RefuseMaxPerShipment = n
	}
	if h := config.VP.GetFloat64("random_events.rest_after_hours"); h > 0 {
		cfg.RestAfterHours = h
	}
	if m := config.VP.GetFloat64("random_events.rest_minutes"); m > 0 {
		cfg.RestMinutes = m
	}
	if p := config.VP.GetFloat64("random_events.weather_prob_per_tick"); p > 0 {
		cfg.WeatherProbPerTick = p
	}
	if ws := getFloat64SliceCfg("random_events.weather_delay_seconds"); len(ws) >= 2 && ws[0] > 0 && ws[1] >= ws[0] {
		cfg.WeatherDelaySeconds = [2]float64{ws[0], ws[1]}
	}
	if p := config.VP.GetFloat64("random_events.accident_prob_per_tick"); p > 0 {
		cfg.AccidentProbPerTick = p
	}
	if ms := getFloat64SliceCfg("random_events.maintenance_seconds"); len(ms) >= 2 && ms[0] > 0 && ms[1] >= ms[0] {
		cfg.MaintenanceSeconds = [2]float64{ms[0], ms[1]}
	}
	if lf := getFloat64SliceCfg("random_events.level_factor"); len(lf) == 5 {
		copy(cfg.LevelFactor[:], lf)
	}
	randomEventCfg = cfg
	Logger.Logger.Info("随机事件配置已加载", zap.Bool("enabled", cfg.Enabled), zap.Float64("refuseProb", cfg.DriverRefuseProb), zap.Float64("accidentProb", cfg.AccidentProbPerTick))
}

// levelFactor 根据司机等级返回概率修正系数(等级越高考评越稳，修正系数越小)
func levelFactor(level int) float64 {
	return levelFactorOf(randomEventCfg, level)
}

// levelFactorOf 按指定配置取司机等级修正系数（实验运行时使用 rt.eventCfg）
func levelFactorOf(cfg RandomEventConfig, level int) float64 {
	if level < 1 {
		level = 1
	}
	if level > 5 {
		level = 5
	}
	return cfg.LevelFactor[level-1]
}

// randomRangeSeconds 区间内取整秒。draw ∈ [0,1)，由调用方提供（实验种子 RNG 或测试随机源）。
func randomRangeSeconds(rng [2]float64, draw float64) int {
	if rng[1] <= rng[0] {
		return int(rng[0])
	}
	return int(rng[0] + draw*(rng[1]-rng[0]))
}

// recordRandomEvent 落库一条随机事件记录（虚拟时刻 + 实验维度），失败仅告警不影响主流程。
func (rt *ExperimentRuntime) recordRandomEvent(eventType string, vehicleID uint, driverID uint, shipmentID uint, taskID uint, severity int, durationSeconds int, description string, detail string, now time.Time) {
	err := repository.CreateRandomEvent(&model.RandomEvent{
		EventTime:       now,
		EventType:       eventType,
		VehicleId:       vehicleID,
		DriverId:        driverID,
		ShipmentId:      shipmentID,
		TaskId:          taskID,
		Severity:        severity,
		DurationSeconds: durationSeconds,
		Description:     description,
		Detail:          detail,
		Status:          "active",
		ExperimentId:    rt.Exp.Id,
	})
	if err != nil {
		Logger.Logger.Warn("记录随机事件失败", zap.String("eventType", eventType), zap.Error(err))
	}
}

// isVehiclePaused 判断车辆在指定（虚拟）时刻是否处于暂停窗口内(休息/极端天气等)，暂停期间不推进。
func isVehiclePaused(v *model.Vehicle, now time.Time) bool {
	if v == nil || v.PauseUntil == nil {
		return false
	}
	return now.Before(*v.PauseUntil)
}

// EnsureDriversForVehicles 兼容迁移：为尚未绑定司机的模板车辆(experiment_id=0)自动生成一名默认司机。
// 实验实例车辆(experiment_id>0)的司机由运行时实例化创建，不在此处理。
func EnsureDriversForVehicles() {
	vehicles, err := repository.ListVehiclesByExp(0, 0)
	if err != nil {
		Logger.Logger.Error("兼容迁移：查询车辆列表失败", zap.Error(err))
		return
	}
	created := 0
	for _, v := range vehicles {
		if v == nil {
			continue
		}
		if v.DriverId > 0 {
			continue
		}
		if _, err := repository.GetDriverByVehicle(v.Id); err == nil {
			continue
		}
		level := 2 + rand.Intn(4) // 2~5
		driver := &model.Driver{
			Name:             "司机" + intToStrUint(v.Id),
			Phone:            "1380000" + intToStrUint(v.Id),
			LicenseNo:        "D" + intToStrUint(v.Id*1000),
			Level:            level,
			Fatigue:          0,
			ContinuousHours:  0,
			Status:           1,
			RefuseCount:      0,
			VehicleId:        v.Id,
			UpdateTime:       time.Now(),
		}
		if err := repository.CreateDriver(driver); err != nil {
			Logger.Logger.Warn("兼容迁移：创建司机失败", zap.Uint("vehicleId", v.Id), zap.Error(err))
			continue
		}
		_ = repository.UpdateVehicleFields(int(v.Id), map[string]interface{}{"driver_id": driver.Id})
		created++
	}
	Logger.Logger.Info("兼容迁移：司机绑定完成", zap.Int("created", created), zap.Int("total", len(vehicles)))
}

func intToStrUint(v uint) string {
	if v == 0 {
		return "0"
	}
	b := make([]byte, 0, 12)
	for v > 0 {
		b = append([]byte{byte(v%10) + '0'}, b...)
		v /= 10
	}
	return string(b)
}

// accumulateDriverFatigue 每推进一段里程累计司机连续驾驶时长，达到阈值触发强制休息。
// 休息期间车辆暂停移动(pause_until)，司机标记休息中。全部使用虚拟时钟与实验有效配置（rt.eventCfg）。
func accumulateDriverFatigue(rt *ExperimentRuntime, task *model.OrderTask, v *model.Vehicle, durationHour float64) {
	cfg := rt.eventCfg
	if !cfg.Enabled || v == nil || task == nil || durationHour <= 0 {
		return
	}
	driver, err := repository.GetDriverByVehicle(v.Id)
	if err != nil || driver == nil {
		return
	}
	now := rt.Clock.Now()
	// 已在休息窗口内的司机不重复累计
	if driver.RestUntil != nil && now.Before(*driver.RestUntil) {
		return
	}

	newHours := driver.ContinuousHours + durationHour
	if newHours >= cfg.RestAfterHours {
		restSeconds := int(cfg.RestMinutes * 60)
		restUntil := now.Add(time.Duration(restSeconds) * time.Second)
		_ = repository.UpdateDriverFields(driver.Id, map[string]interface{}{
			"rest_until":       restUntil,
			"status":           2,
			"continuous_hours": 0, // 休息后重置
			"fatigue":          1.0,
			"update_time":      now,
		})
		// 车辆同步暂停
		_ = repository.UpdateVehicleFields(int(v.Id), map[string]interface{}{"pause_until": restUntil})
		rt.recordRandomEvent("rest", v.Id, driver.Id, task.ShipmentId, task.Id, 1, restSeconds, "司机连续驾驶达到阈值，强制休息", "", now)
		Logger.Logger.Warn("随机事件：司机疲劳需强制休息", zap.Uint("vehicleId", v.Id), zap.Uint("driverId", driver.Id), zap.Float64("hours", newHours))
		return
	}

	fatigue := newHours / cfg.RestAfterHours
	if fatigue > 1 {
		fatigue = 1
	}
	_ = repository.UpdateDriverFields(driver.Id, map[string]interface{}{
		"continuous_hours": newHours,
		"fatigue":          fatigue,
		"update_time":      now,
	})
}

// maybeRefuseAssignment 发车前判定司机是否拒绝接本单。
// 返回 true 表示已拒绝并完成回退(删除任务/释放载重/运单回退 Sleeping)，调用方应放弃发车。
// 同一运单被拒次数达到上限时强制接受，防止死循环。
func maybeRefuseAssignment(rt *ExperimentRuntime, task *model.OrderTask, v *model.Vehicle, shipment *model.Shipment) bool {
	cfg := rt.eventCfg
	if !cfg.Enabled || task == nil || v == nil || shipment == nil {
		return false
	}

	driver, err := repository.GetDriverByVehicle(v.Id)
	if err != nil || driver == nil {
		return false
	}

	refuseCount, _ := repository.CountRefuseByShipment(shipment.Id)
	if refuseCount >= int64(cfg.RefuseMaxPerShipment) {
		return false
	}

	if rt.Draw() >= cfg.DriverRefuseProb*levelFactorOf(cfg, driver.Level) {
		return false
	}

	// 拒绝：删除 Planned 任务
	if err := repository.DeleteOrderTask(int(task.Id)); err != nil {
		Logger.Logger.Warn("拒单回退：删除任务失败", zap.Uint("taskId", task.Id), zap.Error(err))
		return false
	}

	// 释放车辆预占载重
	cargo, _ := repository.GetCargo(int(shipment.CargoId))
	if cargo != nil {
		load := cargo.Weight * shipment.Count
		newReserved := v.ReservedSize - load
		if newReserved < 0 {
			newReserved = 0
		}
		_ = repository.UpdateVehicleReservedSize(int(v.Id), newReserved)
	}

	// 运单回退 Sleeping，等待下一轮 CreateOrderTask 重新分配
	_ = repository.UpdateShipmentStatus(int(shipment.Id), constant.ShipmentStatusSleeping, constant.ShipmentStatusQueued)

	// 司机拒单次数累计
	_ = repository.UpdateDriverFields(driver.Id, map[string]interface{}{"refuse_count": driver.RefuseCount + 1, "update_time": rt.Clock.Now()})

	// 车辆若因此单进入 Ready 且无其他 Planned 任务，则回退为空闲
	if v.Status == constant.VehicleStatusReady {
		if _, err := repository.GetFirstTaskByVehicleAndSequential(v.Id, constant.OrderTaskSequentialPlanned); err != nil {
			_ = repository.UpdateVehicleStatus(int(v.Id), constant.VehicleStatusFree, constant.VehicleStatusReady)
		}
	}

	rt.recordRandomEvent("refuse", v.Id, driver.Id, shipment.Id, task.Id, 1, 0, "司机拒绝接单，运单回退待重新分配", "", rt.Clock.Now())
	Logger.Logger.Warn("随机事件：司机拒绝接单", zap.Uint("vehicleId", v.Id), zap.Uint("driverId", driver.Id), zap.Uint("shipmentId", shipment.Id))
	return true
}