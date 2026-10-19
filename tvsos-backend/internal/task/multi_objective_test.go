package task

import (
	"testing"
	"time"

	"github.com/kiritosuki/mover/internal/constant"
	"github.com/kiritosuki/mover/internal/model"
	sandboxpkg "github.com/kiritosuki/mover/internal/sandbox"
)

// ---- 纯函数层：帕累托支配 / 标量化 / 档案 / 上下文 ----

func TestParetoDominance(t *testing.T) {
	a := ObjectiveVector{C2UnitCost: 1.0, U2LoadingRate: 0.6, U4EmptyRate: 0.2, T2ResponseP95: 100}
	b := ObjectiveVector{C2UnitCost: 2.0, U2LoadingRate: 0.5, U4EmptyRate: 0.3, T2ResponseP95: 200}
	if dom := paretoDominance(a, b); dom != 1 {
		t.Fatalf("a 应支配 b（全维更优），got %d", dom)
	}
	if dom := paretoDominance(b, a); dom != -1 {
		t.Fatalf("b 应被 a 支配，got %d", dom)
	}
	// 互不支配：一好一坏
	c := ObjectiveVector{C2UnitCost: 0.5, U2LoadingRate: 0.4, U4EmptyRate: 0.2, T2ResponseP95: 100}
	if dom := paretoDominance(a, c); dom != 0 {
		t.Fatalf("a/c 互不支配，got %d", dom)
	}
	// 相等向量互不支配
	if dom := paretoDominance(a, a); dom != 0 {
		t.Fatalf("相等向量应互不支配，got %d", dom)
	}
	// U2 方向验证：装载率更高者支配（其余相等）
	d := ObjectiveVector{C2UnitCost: 1.0, U2LoadingRate: 0.8, U4EmptyRate: 0.2, T2ResponseP95: 100}
	if dom := paretoDominance(d, a); dom != 1 {
		t.Fatalf("U2 更高应支配，got %d", dom)
	}
}

func TestObjectiveContextSoftBudgets(t *testing.T) {
	stats := &Statistics{}
	stats.UnitCost = 10.0
	stats.UtilizationStats.LoadingRateAvg = 0.5
	stats.EmptyDistStats.EmptyRate = 0.3
	stats.MatchResponseStats.P95 = 900

	// primary=C2：U2/U4/T2 走自适应（±容差），F1 缺省绝对预算 0.4，C2 自身不约束
	params := sandboxpkg.ObjectiveParams{
		Mode:      sandboxpkg.ObjectiveModePrimary,
		Primary:   sandboxpkg.ObjectivePrimaryC2,
		Tolerance: 0.2,
		Budgets:   map[string]float64{"F1_gini_max": 0.4},
	}
	ctx := newObjectiveContext(params, stats)
	if _, ok := ctx.softBudgets["C2"]; ok {
		t.Fatal("主目标自身不应有软约束")
	}
	if v, ok := ctx.softBudgets["F1"]; !ok || v != 0.4 {
		t.Fatalf("F1 绝对预算应=0.4, got %v %v", v, ok)
	}
	// U2 下限 = 0.5×(1-0.2) = 0.4
	if v, ok := ctx.softBudgets["U2"]; !ok || v < 0.399 || v > 0.401 {
		t.Fatalf("U2 自适应预算应≈0.4, got %v %v", v, ok)
	}
	// U4 上限 = 0.3×1.2 = 0.36
	if v, ok := ctx.softBudgets["U4"]; !ok || v < 0.359 || v > 0.361 {
		t.Fatalf("U4 自适应预算应≈0.36, got %v %v", v, ok)
	}
	// T2 上限 = 900×1.2 = 1080
	if v, ok := ctx.softBudgets["T2"]; !ok || v < 1079 || v > 1081 {
		t.Fatalf("T2 自适应预算应≈1080, got %v %v", v, ok)
	}

	// 绝对预算优先于自适应；主目标切换后旧主目标恢复约束
	params2 := sandboxpkg.ObjectiveParams{
		Mode:      sandboxpkg.ObjectiveModePrimary,
		Primary:   sandboxpkg.ObjectivePrimaryU2,
		Tolerance: 0.15,
		Budgets:   map[string]float64{"U4_max": 0.5},
	}
	ctx2 := newObjectiveContext(params2, stats)
	if v, ok := ctx2.softBudgets["U2"]; ok {
		t.Fatalf("主目标 U2 自身不应约束, got %v", v)
	}
	if v, ok := ctx2.softBudgets["U4"]; !ok || v != 0.5 {
		t.Fatalf("U4 绝对预算应=0.5（优先于自适应），got %v %v", v, ok)
	}
	if v, ok := ctx2.softBudgets["C2"]; !ok || v < 11.4 || v > 11.6 {
		t.Fatalf("C2 自适应预算应≈11.5, got %v %v", v, ok)
	}
}

func TestViolationAndScore(t *testing.T) {
	params := sandboxpkg.ObjectiveParams{
		Mode:    sandboxpkg.ObjectiveModePrimary,
		Primary: sandboxpkg.ObjectivePrimaryU4,
	}
	ctx := newObjectiveContext(params, nil)
	ctx.calibratePrimaryScale(ObjectiveVector{U4EmptyRate: 0.4})

	// U4 主项 + F1 违反惩罚：F1=0.6 超预算 0.4 → 相对违反 0.5
	vec := ObjectiveVector{U4EmptyRate: 0.4, F1Gini: 0.6}
	ctx.softBudgets["F1"] = 0.4
	viol := ctx.violationOf(vec)
	if viol < 0.49 || viol > 0.51 {
		t.Fatalf("F1 相对违反量应≈0.5, got %v", viol)
	}
	score := ctx.score(0, vec, 0, 0)
	if score <= ctx.primaryTerm(vec) {
		t.Fatalf("含违反的分数应高于纯主项: score=%v primaryTerm=%v", score, ctx.primaryTerm(vec))
	}

	// 可满足解违反量=0
	okVec := ObjectiveVector{U4EmptyRate: 0.2, F1Gini: 0.3}
	if v := ctx.violationOf(okVec); v != 0 {
		t.Fatalf("可满足解违反量应为0, got %v", v)
	}
	// 不可行解一票否决
	if s := ctx.score(0, okVec, 1, 0); s < SAPenaltyInf {
		t.Fatalf("不可行解分数应含 SAPenaltyInf, got %v", s)
	}
	// 主项方向：U4 越小分数越低
	better := ObjectiveVector{U4EmptyRate: 0.1, F1Gini: 0.3}
	if ctx.primaryTerm(better) >= ctx.primaryTerm(okVec) {
		t.Fatalf("U4 更优的解主项应更低: %v vs %v", ctx.primaryTerm(better), ctx.primaryTerm(okVec))
	}
	// U2 最大化方向主项
	ctx2 := newObjectiveContext(sandboxpkg.ObjectiveParams{Primary: sandboxpkg.ObjectivePrimaryU2}, nil)
	ctx2.calibratePrimaryScale(ObjectiveVector{U2LoadingRate: 0.5})
	if p := ctx2.primaryTerm(ObjectiveVector{U2LoadingRate: 0.8}); p >= 0 {
		t.Fatalf("U2 超基准的主项应为负, got %v", p)
	}
}

func TestParetoArchive(t *testing.T) {
	archive := newParetoArchive("C2")
	archive.offer([]int{0}, stateEvaluation{Vector: ObjectiveVector{C2UnitCost: 2, U2LoadingRate: 0.5}, Score: 2})
	archive.offer([]int{1}, stateEvaluation{Vector: ObjectiveVector{C2UnitCost: 3, U2LoadingRate: 0.4}, Score: 3}) // 被支配，不入档
	if archive.size() != 1 {
		t.Fatalf("被支配解不应入档, size=%d", archive.size())
	}
	archive.offer([]int{2}, stateEvaluation{Vector: ObjectiveVector{C2UnitCost: 1, U2LoadingRate: 0.6}, Score: 1}) // 支配第一个（C2与U2双优）
	if archive.size() != 1 {
		t.Fatalf("应剔除被支配成员, size=%d", archive.size())
	}
	if entry, ok := archive.bestByScore(); !ok || entry.assigned[0] != 2 {
		t.Fatalf("档案最优应保留 entry2, got %+v", entry)
	}
	// 容量淘汰：填充超过上限
	for i := 0; i < paretoArchiveCap+20; i++ {
		vec := ObjectiveVector{C2UnitCost: float64(1000 - i), U2LoadingRate: 0.1, U4EmptyRate: 0.5, T2ResponseP95: 100}
		archive.offer([]int{i}, stateEvaluation{Vector: vec, Score: float64(i)})
	}
	if archive.size() > paretoArchiveCap {
		t.Fatalf("档案规模应≤%d, got %d", paretoArchiveCap, archive.size())
	}
}

// ---- SA 集成层：主目标切换改变指派倾向 ----

func multiObjectiveFixture() ([]*model.Vehicle, []*model.Shipment, map[uint]*model.Cargo, map[uint]int, map[uint]*model.Poi) {
	now := time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC)
	// 两辆同位车：小车容量 1000、大车容量 10000（装载率对指派敏感）
	vehicles := []*model.Vehicle{
		{Id: 1, Lon: 104.0, Lat: 30.5, Speed: 30, Status: constant.VehicleStatusFree, UpdateTime: now.Add(-time.Minute), Capacity: 1000, Tybe: 1},
		{Id: 2, Lon: 104.0, Lat: 30.5, Speed: 30, Status: constant.VehicleStatusFree, UpdateTime: now.Add(-time.Minute), Capacity: 10000, Tybe: 1},
	}
	// 三笔同货类订单，单笔 100kg
	shipments := []*model.Shipment{
		{Id: 1, Status: constant.ShipmentStatusSleeping, CreateTime: now.Add(-time.Minute), CargoId: 1, Count: 1, StartPoiId: 1, EndPoiId: 2},
		{Id: 2, Status: constant.ShipmentStatusSleeping, CreateTime: now.Add(-time.Minute), CargoId: 1, Count: 1, StartPoiId: 1, EndPoiId: 2},
		{Id: 3, Status: constant.ShipmentStatusSleeping, CreateTime: now.Add(-time.Minute), CargoId: 1, Count: 1, StartPoiId: 1, EndPoiId: 2},
	}
	cargos := map[uint]*model.Cargo{1: {Id: 1, Weight: 100, Tybe: 1}}
	pois := map[uint]*model.Poi{
		1: {Id: 1, Lon: 104.0, Lat: 30.5},
		2: {Id: 2, Lon: 104.02, Lat: 30.51},
	}
	loads := map[uint]int{1: 0, 2: 0}
	return vehicles, shipments, cargos, loads, pois
}

func TestSAPrimaryObjectiveSwitchChangesAssignment(t *testing.T) {
	vehicles, shipments, cargos, loads, pois := multiObjectiveFixture()
	now := time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC)
	stats := buildStatistics(vehicles, nil, shipments, cargos, pois, now)

	// 场景一：主目标 U2（装载率）→ 倾向把货并给小车（装载率更高）
	rtU2 := newStubRuntime()
	rtU2.Params.Objectives = sandboxpkg.ObjectiveParams{
		Mode: sandboxpkg.ObjectiveModePrimary, Primary: sandboxpkg.ObjectivePrimaryU2, Tolerance: 0.15,
	}
	ctxU2 := newObjectiveContext(rtU2.Params.Objectives, stats)
	saU2 := assignBySimulatedAnnealing(rtU2, ctxU2, shipments, vehicles, stats, cargos, cloneProjectedLoads(loads), pois)
	if len(saU2) != 3 {
		t.Fatalf("U2 模式应完成全部指派, got %d", len(saU2))
	}
	smallVehicleCount := 0
	for _, v := range saU2 {
		if v != nil && v.Id == 1 {
			smallVehicleCount++
		}
	}
	if smallVehicleCount < 2 {
		t.Fatalf("主目标 U2 应倾向把订单集中给小容量车辆(装载率更高)，小车指派数=%d/3", smallVehicleCount)
	}

	// 场景二：weighted 模式 → 行为与旧口径一致（无主目标概念，仅成本）
	rtW := newStubRuntime()
	rtW.Params.Objectives = sandboxpkg.ObjectiveParams{Mode: sandboxpkg.ObjectiveModeWeighted}
	ctxW := newObjectiveContext(rtW.Params.Objectives, stats)
	saW := assignBySimulatedAnnealing(rtW, ctxW, shipments, vehicles, stats, cargos, cloneProjectedLoads(loads), pois)
	if len(saW) != 3 {
		t.Fatalf("weighted 模式应完成全部指派, got %d", len(saW))
	}

	// 场景三：多目标运行统计已被记录（批次/前沿）；weighted 后跑 → 最近批次为 weighted 且无主目标
	view := GetObjectiveRunStats()
	if view.Batches < 2 {
		t.Fatalf("多目标运行统计应记录至少 2 个批次, got %d", view.Batches)
	}
	if view.Mode != sandboxpkg.ObjectiveModeWeighted || view.Primary != "" {
		t.Fatalf("最近批次统计应为 weighted 模式(无主目标)，got mode=%s primary=%q", view.Mode, view.Primary)
	}
}

func TestSAConvergenceRecordsObjectiveFields(t *testing.T) {
	vehicles, shipments, cargos, loads, pois := multiObjectiveFixture()
	now := time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC)
	stats := buildStatistics(vehicles, nil, shipments, cargos, pois, now)

	rt := newStubRuntime()
	// 缩小 SA 规模加快测试
	rt.Params.Algorithm.SAItersPerTemp = 5
	rt.Params.Algorithm.SAInitialTemp = 100
	ctx := newObjectiveContext(rt.Params.Objectives, stats)
	_ = assignBySimulatedAnnealing(rt, ctx, shipments, vehicles, stats, cargos, cloneProjectedLoads(loads), pois)

	rt.convergenceMu.Lock()
	curve := rt.convergence
	rt.convergenceMu.Unlock()
	if len(curve) == 0 {
		t.Fatal("收敛曲线不应为空")
	}
	last := curve[len(curve)-1]
	if last.ParetoSize < 1 {
		t.Fatalf("primary 模式收敛点应含帕累托档案规模, got %+v", last)
	}
	if last.BestCost <= 0 {
		t.Fatalf("ε-约束分数应为正, got %+v", last)
	}
}

func TestObjectiveRunStatsReset(t *testing.T) {
	ResetObjectiveRunStats()
	if v := GetObjectiveRunStats(); v.Batches != 0 || v.FrontMax != 0 || v.LastFront != nil {
		t.Fatalf("重置后统计应为零值, got %+v", v)
	}
	recordObjectiveBatch(nil, stateEvaluation{Violation: 0.2, Vector: ObjectiveVector{C2UnitCost: 1}}, 3,
		[]ObjectiveVector{{C2UnitCost: 1}, {C2UnitCost: 2}})
	v := GetObjectiveRunStats()
	if v.Batches != 1 || v.FeasibleBatches != 0 || v.FrontMax != 3 || v.FrontAvg != 3 {
		t.Fatalf("统计累计不符: %+v", v)
	}
	if len(v.LastFront) != 2 {
		t.Fatalf("最近前沿应保留 2 个解, got %d", len(v.LastFront))
	}
}
