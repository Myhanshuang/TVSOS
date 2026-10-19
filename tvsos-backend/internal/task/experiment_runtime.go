package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/kiritosuki/mover/internal/constant"
	"github.com/kiritosuki/mover/internal/model"
	"github.com/kiritosuki/mover/internal/realtime"
	"github.com/kiritosuki/mover/internal/repository"
	sandboxpkg "github.com/kiritosuki/mover/internal/sandbox"
	"go.uber.org/zap"
)

// ============================================================================
// ExperimentRuntime 实验运行时（沙箱系统核心）
//
// 单实验串行模型：全系统同一时刻只有一个活跃 ExperimentRuntime（currentRuntime）。
// 运行时为严格复现而设计：
//  1. 虚拟时钟：所有时间语义（暂停截止/阶段时间戳/事件时刻/快照时刻）均为虚拟时间；
//  2. 种子 RNG：SA 调度、司机等级、拒单判定等全部随机源来自 rt.rng（seed=experiment.seed）；
//  3. 单线程 tick：主循环每个虚拟 tick 顺序执行 释放订单→派单→消费事件→恢复→移动→快照，
//     RNG 消费顺序完全确定，同种子同参数 ⇒ 逐 tick 一致；
//  4. 预生成脚本：订单流与天气/车祸事件由数据生成器预先落库，运行期只消费不摇号。
// ============================================================================

// ExperimentRuntime 一次实验的运行实例
type ExperimentRuntime struct {
	Exp    *model.Experiment
	Sb     *model.Sandbox
	Rules  sandboxpkg.WorldRules
	Params sandboxpkg.ExperimentParams

	rng   *rand.Rand
	rngMu sync.Mutex

	Clock   *VirtualClock
	eventCfg RandomEventConfig // 有效随机事件规则（实验覆盖优先，回落世界规则）

	speedBits atomic.Uint64 // 时间加速比（float64 bits）

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	convergenceMu sync.Mutex
	convergence   []ConvergencePoint // SA 收敛曲线（算法性能记录，对齐 PDF 结果层）
}

// ConvergencePoint SA 收敛曲线采样点。
// BestCost：weighted 模式=原始综合成本；primary 模式=ε-约束标量化分数。
// BestPrimary/BestViolation/ParetoSize 为多目标模式扩展字段（omitempty 兼容旧数据）。
type ConvergencePoint struct {
	Batch         int64   `json:"batch"`
	Temp          float64 `json:"temp"`
	BestCost      float64 `json:"bestCost"`
	BestPrimary   float64 `json:"bestPrimary,omitempty"`
	BestViolation float64 `json:"bestViolation,omitempty"`
	ParetoSize    int     `json:"paretoSize,omitempty"`
}

// ExpID 实验 id（便捷方法）
func (rt *ExperimentRuntime) ExpID() uint { return rt.Exp.Id }

// Draw 从实验种子 RNG 取随机数（互斥保护；单线程主循环下顺序确定）
func (rt *ExperimentRuntime) Draw() float64 {
	rt.rngMu.Lock()
	defer rt.rngMu.Unlock()
	return rt.rng.Float64()
}

// DrawIntn 从实验种子 RNG 取 [0,n) 整数
func (rt *ExperimentRuntime) DrawIntn(n int) int {
	rt.rngMu.Lock()
	defer rt.rngMu.Unlock()
	return rt.rng.Intn(n)
}

// GetSpeed 当前时间加速比
func (rt *ExperimentRuntime) GetSpeed() float64 {
	s := math.Float64frombits(rt.speedBits.Load())
	if s <= 0 {
		return 1.0
	}
	return s
}

// SetSpeed 设置时间加速比（只改变真实世界 tick 频率，不影响虚拟时间序列）
func (rt *ExperimentRuntime) SetSpeed(multiplier float64) {
	if multiplier <= 0 {
		return
	}
	rt.speedBits.Store(math.Float64bits(multiplier))
}

// SpeedKmhAt 交通环境模型：某虚拟时刻的道路有效时速 = 基准速度 × 时段系数（确定性）。
func (rt *ExperimentRuntime) SpeedKmhAt(virtualNow time.Time) float64 {
	base := rt.Rules.Geography.AvgSpeedKmh
	if base <= 0 {
		base = 35
	}
	hour := virtualNow.Hour()
	for _, h := range rt.Rules.Traffic.RushHours {
		if h == hour {
			f := rt.Rules.Traffic.RushSpeedFactor
			if f > 0 {
				return base * f
			}
		}
	}
	return base
}

// RecordConvergence 记录一个 SA 收敛采样点
func (rt *ExperimentRuntime) RecordConvergence(p ConvergencePoint) {
	rt.convergenceMu.Lock()
	rt.convergence = append(rt.convergence, p)
	rt.convergenceMu.Unlock()
}

// ---------------- 运行时管理器（单实验串行） ----------------

var (
	runtimeMu     sync.Mutex
	currentRt     *ExperimentRuntime
	activeExpID   atomic.Uint64
)

// CurrentRuntime 当前活跃实验运行时（无则 nil）
func CurrentRuntime() *ExperimentRuntime {
	runtimeMu.Lock()
	defer runtimeMu.Unlock()
	return currentRt
}

// CurrentExperimentID 当前活跃实验 id（0=无）
func CurrentExperimentID() uint {
	return uint(activeExpID.Load())
}

// fleetTemplateVehicle 车队模板车辆定义（与迁移脚本同款 JSON）
type fleetTemplateVehicle struct {
	License  string  `json:"license"`
	Tybe     int     `json:"tybe"`
	Size     int     `json:"size"`
	Capacity int     `json:"capacity"`
	Length   float64 `json:"length"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	Lon      float64 `json:"lon"`
	Lat      float64 `json:"lat"`
}

// StartExperiment 启动实验：实例化车队/司机 → 建运行时 → 主循环。
// 要求实验状态为 ready（数据已生成）；单实验串行，已有运行中实验时拒绝。
func StartExperiment(experimentID uint) error {
	runtimeMu.Lock()
	defer runtimeMu.Unlock()
	if currentRt != nil {
		return errors.New("已有运行中的实验，请先停止")
	}

	exp, err := repository.GetExperiment(experimentID)
	if err != nil {
		return fmt.Errorf("实验不存在: %w", err)
	}
	if exp.Status != model.ExperimentStatusReady {
		return fmt.Errorf("实验状态为 %s，须为 ready（请先生成数据）", exp.Status)
	}
	sb, err := repository.GetSandbox(exp.SandboxId)
	if err != nil {
		return fmt.Errorf("沙箱不存在: %w", err)
	}
	rules, err := sandboxpkg.ParseWorldRules(sb.WorldRulesToml)
	if err != nil {
		return fmt.Errorf("世界规则解析失败: %w", err)
	}
	params, err := sandboxpkg.ParseExperimentParams(exp.ParamsToml)
	if err != nil {
		return fmt.Errorf("实验参数解析失败: %w", err)
	}
	params.Seed = exp.Seed

	rt := &ExperimentRuntime{
		Exp:      exp,
		Sb:       sb,
		Rules:    rules,
		Params:   params,
		rng:      rand.New(rand.NewSource(exp.Seed)),
		Clock:    NewVirtualClock(rules.Time.StartTime, rules.Time.TickSeconds, params.Simulation.DurationHours),
		eventCfg: effectiveEventConfig(params, rules),
	}
	rt.speedBits.Store(math.Float64bits(params.Simulation.Speed))
	rt.ctx, rt.cancel = context.WithCancel(context.Background())

	// 实例化本实验专属车队与司机（种子 RNG 决定司机等级，确定性）
	if err := rt.instantiateFleet(); err != nil {
		return fmt.Errorf("车队实例化失败: %w", err)
	}

	// 经济环境接入成本计算（对齐 PDF 经济环境变量）
	SetEconomyRules(&rules.Economy)

	// 清理运行态路径缓存；重置进程内指标累计器（新实验新窗口）
	resetRuntimeRouteCaches()
	ResetSolverTiming()
	ResetMatchingQuality()
	ResetOptimizationGap()
	ResetObjectiveRunStats()
	resetDashboardTrends()

	now := time.Now()
	if err := repository.UpdateExperimentFields(exp.Id, map[string]interface{}{
		"status":     model.ExperimentStatusRunning,
		"started_at": now,
		"progress":   0,
	}); err != nil {
		return err
	}
	exp.Status = model.ExperimentStatusRunning
	exp.StartedAt = &now

	currentRt = rt
	activeExpID.Store(uint64(exp.Id))
	RecordSystemEvent(exp.Id, "startup", "info", fmt.Sprintf("实验#%d(%s)启动，种子=%d", exp.Id, exp.Name, exp.Seed), "")

	rt.wg.Add(2)
	go rt.masterLoop()
	go rt.statsBroadcastLoop()
	Logger.Logger.Info("实验运行时已启动", zap.Uint("experimentId", exp.Id), zap.Int64("seed", exp.Seed))
	return nil
}

// StopExperiment 停止当前实验。finished=true 表示到达固定虚拟时长正常完成（落实验结果）；
// finished=false 表示人工中止（状态 aborted，不写结果）。
func StopExperiment(finished bool) error {
	runtimeMu.Lock()
	rt := currentRt
	runtimeMu.Unlock()
	if rt == nil {
		return errors.New("当前没有运行中的实验")
	}
	rt.cancel()
	rt.wg.Wait()

	runtimeMu.Lock()
	currentRt = nil
	activeExpID.Store(0)
	runtimeMu.Unlock()

	SetEconomyRules(nil)
	rt.finalize(finished)
	return nil
}

// AbortStaleExperiments 服务启动时调用：把残留的 running/generating 实验标记为 aborted/failed。
// 约定：后端重启即中断当前实验（方案 A 不做运行态恢复）。
func AbortStaleExperiments() {
	if err := repository.MarkStaleExperimentsAborted(); err != nil {
		Logger.Logger.Warn("清理残留实验状态失败", zap.Error(err))
	}
}

// instantiateFleet 按沙箱车队模板实例化本实验专属 vehicle/driver 行（experiment_id=expID）。
// 司机等级优先取数据生成期预摇的等级表（experiment.driver_levels，与车祸脚本修正同源）；
// 缺失时回落运行时种子 RNG（兼容旧版生成的实验）。
func (rt *ExperimentRuntime) instantiateFleet() error {
	var template []fleetTemplateVehicle
	if err := json.Unmarshal([]byte(rt.Sb.FleetTemplate), &template); err != nil {
		return fmt.Errorf("车队模板解析失败: %w", err)
	}
	if len(template) == 0 {
		return errors.New("车队模板为空")
	}
	count := rt.Params.Fleet.VehicleCount
	if count <= 0 || count > len(template) {
		count = len(template)
	}
	template = template[:count]

	presetLevels := make(map[string]int)
	if rt.Exp.DriverLevels != "" {
		_ = json.Unmarshal([]byte(rt.Exp.DriverLevels), &presetLevels)
	}
	virtualNow := rt.Clock.Now()
	for _, tv := range template {
		v := &model.Vehicle{
			License:      tv.License,
			Lon:          tv.Lon,
			Lat:          tv.Lat,
			Speed:        0,
			UpdateTime:   virtualNow,
			Status:       constant.VehicleStatusFree,
			Tybe:         tv.Tybe,
			Size:         0,
			ReservedSize: 0,
			Capacity:     tv.Capacity,
			Length:       tv.Length,
			Width:        tv.Width,
			Height:       tv.Height,
			Angle:        90,
			ExperimentId: rt.Exp.Id,
		}
		if err := repository.CreateVehicle(v); err != nil {
			return err
		}
		level, ok := presetLevels[tv.License]
		if !ok || level < 1 || level > 5 {
			level = 2 + rt.DrawIntn(4) // 回落：运行时种子 RNG
		}
		driver := &model.Driver{
			Name:         "司机" + tv.License,
			Phone:        "13800000000",
			LicenseNo:    "D" + tv.License,
			Level:        level,
			Status:       1,
			VehicleId:    v.Id,
			UpdateTime:   virtualNow,
			ExperimentId: rt.Exp.Id,
		}
		if err := repository.CreateDriver(driver); err != nil {
			return err
		}
		_ = repository.UpdateVehicleFields(int(v.Id), map[string]interface{}{"driver_id": driver.Id})
	}
	Logger.Logger.Info("实验车队实例化完成", zap.Uint("experimentId", rt.Exp.Id), zap.Int("vehicles", len(template)))
	return nil
}

// effectiveEventConfig 有效随机事件配置：实验级覆盖优先，回落世界规则。
func effectiveEventConfig(params sandboxpkg.ExperimentParams, rules sandboxpkg.WorldRules) RandomEventConfig {
	re := sandboxpkg.EffectiveRandomEvents(params, rules)
	return RandomEventConfig{
		Enabled:              true,
		DriverRefuseProb:     re.DriverRefuseProb,
		RefuseMaxPerShipment: re.RefuseMaxPerShipment,
		RestAfterHours:       re.RestAfterHours,
		RestMinutes:          re.RestMinutes,
		WeatherProbPerTick:   re.WeatherProbPerTick,
		WeatherDelaySeconds:  re.WeatherDelaySeconds,
		AccidentProbPerTick:  re.AccidentProbPerTick,
		MaintenanceSeconds:   re.MaintenanceSeconds,
		LevelFactor:          re.LevelFactor,
	}
}

// ---------------- 主循环（单线程 tick，严格复现的执行序） ----------------

// masterLoop 实验主循环：每个真实间隔推进一个虚拟 tick。
// 真实间隔 = tickSeconds / speed 秒（speed 只改变真实节奏，不改变虚拟序列）。
func (rt *ExperimentRuntime) masterLoop() {
	defer rt.wg.Done()
	for {
		select {
		case <-rt.ctx.Done():
			return
		default:
		}

		rt.tick()

		if rt.Clock.Expired() {
			Logger.Logger.Info("实验到达固定虚拟仿真时长，正常完成", zap.Uint("experimentId", rt.Exp.Id))
			rt.finishFromLoop()
			return
		}

		interval := time.Duration(float64(rt.Clock.TickSeconds()) / rt.GetSpeed() * float64(time.Second))
		if interval < 100*time.Millisecond {
			interval = 100 * time.Millisecond
		}
		select {
		case <-rt.ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}

// finishFromLoop 主循环内正常完成：先摘除全局引用（幂等），再落结果。
func (rt *ExperimentRuntime) finishFromLoop() {
	rt.cancel()
	runtimeMu.Lock()
	if currentRt == rt {
		currentRt = nil
		activeExpID.Store(0)
	}
	runtimeMu.Unlock()
	SetEconomyRules(nil)
	rt.finalize(true)
}

// tick 单个虚拟 tick 的确定性执行序列（顺序即复现保证）。
func (rt *ExperimentRuntime) tick() {
	rt.Clock.Advance()
	rt.releaseDueOrders()
	rt.dispatchSleepingShipments()
	rt.consumeDueEvents()
	rt.recoverMaintenanceVehicles()
	rt.recoverRestingDrivers()
	rt.moveAll()
	rt.maybeSnapshot()
	rt.maybePersistProgress()
}

// releaseDueOrders 到点订单释放为 shipment（Sleeping），进入待派单池。
func (rt *ExperimentRuntime) releaseDueOrders() {
	now := rt.Clock.Now()
	orders, err := repository.ListDueExperimentOrders(rt.Exp.Id, now)
	if err != nil || len(orders) == 0 {
		return
	}
	ids := make([]uint, 0, len(orders))
	for _, o := range orders {
		shipment := &model.Shipment{
			StartPoiId:   o.StartPoiId,
			EndPoiId:     o.EndPoiId,
			Status:       constant.ShipmentStatusSleeping,
			CargoId:      o.CargoId,
			Count:        o.Count,
			CreateTime:   now,
			UpdateTime:   now,
			ExperimentId: rt.Exp.Id,
		}
		if err := repository.CreateShipment(shipment); err != nil {
			Logger.Logger.Warn("释放预生成订单失败", zap.Uint("orderId", o.Id), zap.Error(err))
			continue
		}
		ids = append(ids, o.Id)
	}
	_ = repository.MarkExperimentOrdersReleased(ids)
}

// dispatchSleepingShipments 扫描本实验待派单运单，SA 批量分配。
func (rt *ExperimentRuntime) dispatchSleepingShipments() {
	shipments, err := repository.GetSleepingShipmentByExp(rt.Exp.Id)
	if err != nil || len(shipments) == 0 {
		return
	}
	doCreateOrderTaskBatch(rt, shipments)
}

// consumeDueEvents 消费到点的预生成事件脚本（天气/车祸）。
func (rt *ExperimentRuntime) consumeDueEvents() {
	now := rt.Clock.Now()
	events, err := repository.ListDueExperimentEvents(rt.Exp.Id, now)
	if err != nil || len(events) == 0 {
		return
	}
	consumed := make([]uint, 0, len(events))
	for _, ev := range events {
		if ev == nil {
			continue
		}
		vehicle, vErr := repository.GetVehicleByLicense(rt.Exp.Id, ev.VehicleLicense)
		if vErr != nil || vehicle == nil {
			Logger.Logger.Warn("脚本事件目标车辆不存在，跳过", zap.String("license", ev.VehicleLicense), zap.String("type", ev.EventType))
			consumed = append(consumed, ev.Id)
			continue
		}
		rt.applyScriptedEvent(ev, vehicle, now)
		consumed = append(consumed, ev.Id)
	}
	_ = repository.MarkExperimentEventsConsumed(consumed)
}

// applyScriptedEvent 应用单条脚本事件（虚拟时间语义）。
func (rt *ExperimentRuntime) applyScriptedEvent(ev *model.ExperimentEvent, vehicle *model.Vehicle, now time.Time) {
	duration := ev.DurationSeconds
	until := now.Add(time.Duration(duration) * time.Second)
	var driverID uint
	if d, err := repository.GetDriverByVehicle(vehicle.Id); err == nil && d != nil {
		driverID = d.Id
	}
	switch ev.EventType {
	case "weather":
		_ = repository.UpdateVehicleFields(int(vehicle.Id), map[string]interface{}{"pause_until": until})
		rt.recordRandomEvent("weather", vehicle.Id, driverID, 0, 0, ev.Severity, duration, "极端天气导致运输延后(脚本)", ev.Detail, now)
	case "accident":
		_ = repository.UpdateVehicleFields(int(vehicle.Id), map[string]interface{}{"pause_until": until})
		if vehicle.Status != constant.VehicleStatusMaintenance {
			_ = repository.UpdateVehicleStatus(int(vehicle.Id), constant.VehicleStatusMaintenance, vehicle.Status)
		}
		rt.recordRandomEvent("accident", vehicle.Id, driverID, 0, 0, ev.Severity, duration, "发生车祸，车辆受损进入维修(脚本)", ev.Detail, now)
		RecordSystemEvent(rt.Exp.Id, "fault", "error", "车辆发生车祸受损，进入维修", "vehicle_license="+ev.VehicleLicense)
	default:
		Logger.Logger.Warn("未知脚本事件类型", zap.String("type", ev.EventType))
	}
}

// recoverMaintenanceVehicles 维修到期恢复（虚拟时间比较）。
func (rt *ExperimentRuntime) recoverMaintenanceVehicles() {
	vehicles, err := repository.ListVehiclesByExp(rt.Exp.Id, constant.VehicleStatusMaintenance)
	if err != nil {
		return
	}
	now := rt.Clock.Now()
	for _, v := range vehicles {
		if v == nil || v.PauseUntil == nil || now.Before(*v.PauseUntil) {
			continue
		}
		newStatus := constant.VehicleStatusFree
		if runningTask, err := repository.GetFirstRunningTaskByVehicle(v.Id); err == nil && runningTask != nil {
			newStatus = constant.VehicleStatusRunning
		} else if planned, err := repository.GetFirstTaskByVehicleAndSequential(v.Id, constant.OrderTaskSequentialPlanned); err == nil && planned != nil {
			newStatus = constant.VehicleStatusReady
		}
		if err := repository.UpdateVehicleStatus(int(v.Id), newStatus, constant.VehicleStatusMaintenance); err != nil {
			continue
		}
		_ = repository.UpdateVehicleFields(int(v.Id), map[string]interface{}{"pause_until": nil})
		if driver, err := repository.GetDriverByVehicle(v.Id); err == nil && driver != nil {
			_ = repository.UpdateDriverFields(driver.Id, map[string]interface{}{"status": 1, "rest_until": nil})
		}
		RecordSystemEvent(rt.Exp.Id, "recover", "info", "车辆维修完成，恢复可用", "vehicle_license="+v.License)
	}
}

// recoverRestingDrivers 司机休息到期恢复（虚拟时间比较，兜底）。
func (rt *ExperimentRuntime) recoverRestingDrivers() {
	drivers, err := repository.ListDriversByExp(rt.Exp.Id, 2)
	if err != nil {
		return
	}
	now := rt.Clock.Now()
	for _, d := range drivers {
		if d == nil || d.RestUntil == nil || now.Before(*d.RestUntil) {
			continue
		}
		_ = repository.UpdateDriverFields(d.Id, map[string]interface{}{"status": 1, "rest_until": nil})
	}
}

// moveAll 顺序推进本实验全部在途任务（单线程确定性：不再按任务起 goroutine）。
func (rt *ExperimentRuntime) moveAll() {
	tasks, err := repository.GetRunningTasksOrderedByVehicleByExp(rt.Exp.Id)
	if err != nil {
		return
	}
	movingVehicles := make(map[uint]bool)
	for _, task := range tasks {
		if task == nil || movingVehicles[task.VehicleId] {
			continue
		}
		movingVehicles[task.VehicleId] = true
		moveVehicle(rt, task)
	}
}

// maybeSnapshot 每 6 个虚拟 tick（默认 1 虚拟分钟）写一次指标快照。
func (rt *ExperimentRuntime) maybeSnapshot() {
	if rt.Clock.Tick()%6 != 0 {
		return
	}
	stats, err := CalculateGlobalStats(rt.Exp.Id, rt.Clock.Now())
	if err != nil || stats == nil {
		return
	}
	_ = repository.CreateMetricSnapshot(&model.MetricSnapshot{
		SnapshotTime:        rt.Clock.Now(),
		CostValue:           stats.TotalCost,
		TotalThroughput:     stats.ThroughputStats.TotalTransportedTons,
		CargoPending:        stats.CargoStats.PendingDemandTons,
		TransportEfficiency: stats.UtilizationStats.ByCapacity,
		TotalWaitingTime:    stats.VehicleWaitStats.Total,
		TotalEmptyMileage:   stats.EmptyDistStats.TotalEmpty,
		UnitCost:            stats.UnitCost,
		EmptyRate:           stats.EmptyDistStats.EmptyRate,
		LoadingRateStdDev:   stats.UtilizationStats.LoadingRateStd,
		MatchResponseP50:    stats.MatchResponseStats.P50,
		MatchResponseP95:    stats.MatchResponseStats.P95,
		MatchResponseP99:    stats.MatchResponseStats.P99,
		GiniCoefficient:     stats.UtilizationStats.GiniCoefficient,
		CarbonTotal:         stats.CarbonStats.TotalCarbon,
		CarbonPerTonKm:      stats.CarbonStats.CarbonPerTonKm,
		ExperimentId:        rt.Exp.Id,
	})
}

// maybePersistProgress 每 12 tick 落一次虚拟时钟进度。
func (rt *ExperimentRuntime) maybePersistProgress() {
	if rt.Clock.Tick()%12 != 0 {
		return
	}
	_ = repository.UpdateExperimentFields(rt.Exp.Id, map[string]interface{}{"progress": rt.Clock.Progress()})
}

// statsBroadcastLoop 实时统计推送（纯展示，每 3 真实秒；不影响仿真状态，与复现无关）。
func (rt *ExperimentRuntime) statsBroadcastLoop() {
	defer rt.wg.Done()
	interval := getStatsPushInterval()
	for {
		select {
		case <-rt.ctx.Done():
			return
		case <-time.After(interval):
		}
		if realtime.GlobalHub().StatsSubscriberCount() == 0 {
			continue
		}
		snapshot, err := BuildRealtimeDashboard(rt)
		if err != nil {
			continue
		}
		realtime.GlobalHub().BroadcastStats("snapshot", snapshot)
	}
}

// finalize 收尾：正常完成时计算终态指标并写 experiment_result（结果数据层）。
func (rt *ExperimentRuntime) finalize(finished bool) {
	now := time.Now()
	status := model.ExperimentStatusAborted
	if finished {
		status = model.ExperimentStatusFinished
	}

	if finished {
		rt.writeExperimentResult()
	}
	_ = repository.UpdateExperimentFields(rt.Exp.Id, map[string]interface{}{
		"status":      status,
		"finished_at": now,
		"progress":    rt.Clock.Progress(),
	})
	RecordSystemEvent(rt.Exp.Id, "shutdown", "info", fmt.Sprintf("实验#%d结束(%s)", rt.Exp.Id, status), "")
	Logger.Logger.Info("实验运行时已停止", zap.Uint("experimentId", rt.Exp.Id), zap.String("status", status))
}

// writeExperimentResult 终态结果落盘：summary 全量 + 主目标冗余 + SA 收敛曲线（对齐 PDF 结果数据层）。
func (rt *ExperimentRuntime) writeExperimentResult() {
	dashboard, err := BuildRealtimeDashboard(rt)
	if err != nil {
		Logger.Logger.Warn("终态指标计算失败，跳过结果落盘", zap.Error(err))
		return
	}
	summaryJSON, _ := json.Marshal(dashboard.Summary)

	// 供应链链条吨位/方量趋势（任务书 1.1：加工损耗的吨位、方量变化趋势）
	if chainStats := buildSupplyChainTrend(rt.Exp.Id); len(chainStats) > 0 {
		var summaryMap map[string]interface{}
		if err := json.Unmarshal(summaryJSON, &summaryMap); err == nil {
			summaryMap["supply_chains"] = chainStats
			summaryJSON, _ = json.Marshal(summaryMap)
		}
	}

	// 主目标映射（指标体系文档：C2 单位成本 / U2 载重利用率 / U4 空驶率 / T2 响应P95 / F1 基尼 / G2 吨公里碳排）
	primary := map[string]interface{}{
		"C2_unit_cost":          dashboard.Summary.UnitCost,
		"U2_loading_rate":       dashboard.Summary.LoadingRateAvg,
		"U4_empty_rate":         dashboard.Summary.EmptyRate,
		"T2_match_response_p95": dashboard.Summary.MatchResponseP95,
		"F1_gini":               dashboard.Summary.GiniCoefficient,
		"G2_carbon_per_ton_km":  dashboard.Summary.CarbonPerTonKm,
		"total_cost":            dashboard.Summary.CostValue,
		"total_distance_km":     dashboard.Summary.TotalEmptyMileage,
		"total_throughput":      dashboard.Summary.TotalThroughput,
		"transport_efficiency":  dashboard.Summary.TransportEfficiency,
		"match_f1":              dashboard.Summary.MatchF1,
		"solve_time_avg_ms":     dashboard.Summary.SolveTimeAvgMs,
		"accident_count":        dashboard.Summary.AccidentCount,
		"refuse_count":          dashboard.Summary.RefuseCount,
		"virtual_ticks":         rt.Clock.Tick(),
	}

	// 多目标优化信息（主要目标法 + 帕累托考量）：主目标代号/终值/软约束满足率/前沿规模
	objView := GetObjectiveRunStats()
	objPrimary := objView.Primary
	if objPrimary == "" {
		objPrimary = rt.Params.Objectives.NormalizedPrimary() // weighted 模式回落参数口径
	}
	primary["objective_mode"] = objView.Mode
	primary["primary_objective"] = objPrimary
	primary["tolerance"] = rt.Params.Objectives.Tolerance
	switch objPrimary {
	case sandboxpkg.ObjectivePrimaryU2:
		primary["primary_objective_value"] = dashboard.Summary.LoadingRateAvg
	case sandboxpkg.ObjectivePrimaryU4:
		primary["primary_objective_value"] = dashboard.Summary.EmptyRate
	case sandboxpkg.ObjectivePrimaryT2:
		primary["primary_objective_value"] = dashboard.Summary.MatchResponseP95
	default:
		primary["primary_objective_value"] = dashboard.Summary.UnitCost
	}
	if objView.Batches > 0 && objView.Mode == sandboxpkg.ObjectiveModePrimary {
		primary["objective_batches"] = objView.Batches
		primary["objective_feasible_batches"] = objView.FeasibleBatches
		primary["objective_feasible_pct"] = math.Round(objView.FeasiblePct*100) / 100
		primary["objective_violation_avg"] = math.Round(objView.ViolationAvg*1e6) / 1e6
		primary["pareto_avg_front"] = math.Round(objView.FrontAvg*100) / 100
		primary["pareto_max_front"] = objView.FrontMax
		primary["pareto_last_front"] = objView.LastFront
	}
	primaryJSON, _ := json.Marshal(primary)

	rt.convergenceMu.Lock()
	convergenceJSON, _ := json.Marshal(rt.convergence)
	rt.convergenceMu.Unlock()

	_ = repository.UpsertResult(&model.ExperimentResult{
		ExperimentId:   rt.Exp.Id,
		SummaryJson:    string(summaryJSON),
		PrimaryMetrics: string(primaryJSON),
		Convergence:    string(convergenceJSON),
	})
}

// chainNodeStat 链条节点级汇总（吨位/方量趋势的最小口径）。
type chainNodeStat struct {
	Label    string  `json:"label"`
	Cargo    string  `json:"cargo"`
	Orders   int     `json:"orders"`
	Tons     float64 `json:"tons"`
	Volume   float64 `json:"volume"`
	YieldTon float64 `json:"yield_ton"` // 相邻上游节点吨位的实际产出率（首节点=1）
}

// chainTrend 一条链的趋势汇总。
type chainTrend struct {
	Name  string          `json:"name"`
	Nodes []chainNodeStat `json:"nodes"`
}

// buildSupplyChainTrend 汇总实验订单脚本中带链路标记的订单：
// 按（链条 × 节点）聚合件数/吨位/方量，并计算相邻节点实际吨位产出率，
// 呈现加工损耗沿链传导的吨位、方量变化趋势。
func buildSupplyChainTrend(experimentID uint) []chainTrend {
	orders, err := repository.ListExperimentOrdersByExperiment(experimentID)
	if err != nil || len(orders) == 0 {
		return nil
	}
	cargos, err := repository.ListCargos()
	if err != nil {
		return nil
	}
	cargoById := make(map[uint]*model.Cargo, len(cargos))
	for _, cg := range cargos {
		cargoById[cg.Id] = cg
	}
	type key struct{ chain, label string }
	type acc struct {
		cargo  string
		orders int
		tons   float64
		volume float64
	}
	accs := make(map[key]*acc)
	seq := make([]key, 0) // 保持链条配置顺序：按首次出现顺序输出
	for _, o := range orders {
		if o.ChainName == "" || o.NodeLabel == "" {
			continue
		}
		k := key{o.ChainName, o.NodeLabel}
		if _, ok := accs[k]; !ok {
			accs[k] = &acc{}
			seq = append(seq, k)
		}
		a := accs[k]
		a.orders++
		if cg := cargoById[o.CargoId]; cg != nil {
			a.cargo = cg.Name
			a.tons += float64(o.Count*cg.Weight) / 1000.0
			a.volume += float64(o.Count) * cg.Volume
		}
	}
	if len(seq) == 0 {
		return nil
	}
	chainNames := make([]string, 0)
	chainNodes := make(map[string][]chainNodeStat)
	for _, k := range seq {
		if _, seen := chainNodes[k.chain]; !seen {
			chainNames = append(chainNames, k.chain)
			chainNodes[k.chain] = make([]chainNodeStat, 0)
		}
		a := accs[k]
		nodes := chainNodes[k.chain]
		yield := 0.0
		if len(nodes) > 0 && nodes[len(nodes)-1].Tons > 0 {
			yield = a.tons / nodes[len(nodes)-1].Tons
		} else {
			yield = 1
		}
		chainNodes[k.chain] = append(nodes, chainNodeStat{
			Label: k.label, Cargo: a.cargo, Orders: a.orders,
			Tons: math.Round(a.tons*100) / 100, Volume: math.Round(a.volume*100) / 100,
			YieldTon: math.Round(yield*1000) / 1000,
		})
	}
	trends := make([]chainTrend, 0, len(chainNames))
	for _, name := range chainNames {
		trends = append(trends, chainTrend{Name: name, Nodes: chainNodes[name]})
	}
	return trends
}
