package task

import (
	"testing"
	"time"

	sandboxpkg "github.com/kiritosuki/mover/internal/sandbox"
)

// ---- 求解器注册表 ----

func TestSolverRegistry(t *testing.T) {
	want := []string{
		sandboxpkg.SolverSA, sandboxpkg.SolverNSGA2, sandboxpkg.SolverNSGA3,
		sandboxpkg.SolverMOEAD, sandboxpkg.SolverSPEA2, sandboxpkg.SolverMOPSO, sandboxpkg.SolverNF,
	}
	for _, name := range want {
		s := GetSolver(name)
		if s == nil || s.Name() != name {
			t.Fatalf("求解器 %s 应已注册", name)
		}
		if s.DisplayName() == "" {
			t.Fatalf("求解器 %s 缺少显示名", name)
		}
	}
	if fallback := GetSolver("no-such-solver"); fallback.Name() != sandboxpkg.SolverSA {
		t.Fatalf("未知代号应回落 SA, got %s", fallback.Name())
	}
}

// ---- 各插件在标准夹具上完整求解 ----

func TestAllSolversProduceCompleteAssignment(t *testing.T) {
	vehicles, shipments, cargos, loads, pois := multiObjectiveFixture()
	now := time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC)
	stats := buildStatistics(vehicles, nil, shipments, cargos, pois, now)

	solvers := []string{
		sandboxpkg.SolverSA, sandboxpkg.SolverNSGA2, sandboxpkg.SolverNSGA3,
		sandboxpkg.SolverMOEAD, sandboxpkg.SolverSPEA2, sandboxpkg.SolverMOPSO, sandboxpkg.SolverNF,
	}
	for _, name := range solvers {
		rt := newStubRuntime()
		rt.Params.Algorithm.Solver = name
		// 缩小种群/代数加快测试
		rt.Params.Algorithm.Population = 8
		rt.Params.Algorithm.Generations = 6
		rt.Params.Algorithm.RefDivisions = 3
		rt.Params.Algorithm.Neighborhood = 4
		rt.Params.Algorithm.ArchiveSize = 8
		if name == sandboxpkg.SolverSA {
			rt.Params.Algorithm.Population = 0
			rt.Params.Algorithm.Generations = 0
			rt.Params.Algorithm.SAInitialTemp = 100
			rt.Params.Algorithm.SAItersPerTemp = 10
		}
		objCtx := newObjectiveContext(rt.Params.Objectives, stats)
		sa := solveDispatch(rt, objCtx, shipments, vehicles, stats, cargos, cloneProjectedLoads(loads), pois)
		if len(sa) != len(shipments) {
			t.Fatalf("%s 应完成全部指派(期望 %d), got %d", name, len(shipments), len(sa))
		}
		for sid, v := range sa {
			if v == nil {
				t.Fatalf("%s 运单 %d 未指派", name, sid)
			}
		}
	}
}

// 同种子复现：种群算法逐次结果一致
func TestSolverDeterminism(t *testing.T) {
	vehicles, shipments, cargos, loads, pois := multiObjectiveFixture()
	now := time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC)
	stats := buildStatistics(vehicles, nil, shipments, cargos, pois, now)

	run := func() map[uint]uint {
		rt := newStubRuntime()
		rt.Params.Algorithm.Solver = sandboxpkg.SolverNSGA2
		rt.Params.Algorithm.Population = 10
		rt.Params.Algorithm.Generations = 8
		objCtx := newObjectiveContext(rt.Params.Objectives, stats)
		out := solveDispatch(rt, objCtx, shipments, vehicles, stats, cargos, cloneProjectedLoads(loads), pois)
		m := make(map[uint]uint, len(out))
		for sid, v := range out {
			m[sid] = v.Id
		}
		return m
	}
	a, b := run(), run()
	for sid, vid := range a {
		if b[sid] != vid {
			t.Fatalf("同种子两次求解结果不一致: shipment %d → %d vs %d", sid, vid, b[sid])
		}
	}
}

// 主目标切换对种群算法生效：NSGA-II + primary=U2 倾向集中装载
func TestNSGA2PrimaryObjectiveSwitch(t *testing.T) {
	vehicles, shipments, cargos, loads, pois := multiObjectiveFixture()
	now := time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC)
	stats := buildStatistics(vehicles, nil, shipments, cargos, pois, now)

	rt := newStubRuntime()
	rt.Params.Algorithm.Solver = sandboxpkg.SolverNSGA2
	rt.Params.Objectives = sandboxpkg.ObjectiveParams{
		Mode: sandboxpkg.ObjectiveModePrimary, Primary: sandboxpkg.ObjectivePrimaryU2, Tolerance: 0.15,
	}
	rt.Params.Algorithm.Population = 12
	rt.Params.Algorithm.Generations = 12
	objCtx := newObjectiveContext(rt.Params.Objectives, stats)
	out := solveDispatch(rt, objCtx, shipments, vehicles, stats, cargos, cloneProjectedLoads(loads), pois)
	small := 0
	for _, v := range out {
		if v != nil && v.Id == 1 {
			small++
		}
	}
	if small < 2 {
		t.Fatalf("NSGA-II 主目标 U2 应倾向集中给小容量车辆, got %d/3", small)
	}
}

// weighted 模式 + 种群算法 → 派发器回落 SA，不崩溃且可完整指派
func TestWeightedModeFallsBackToSA(t *testing.T) {
	vehicles, shipments, cargos, loads, pois := multiObjectiveFixture()
	now := time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC)
	stats := buildStatistics(vehicles, nil, shipments, cargos, pois, now)

	rt := newStubRuntime()
	rt.Params.Objectives = sandboxpkg.ObjectiveParams{Mode: sandboxpkg.ObjectiveModeWeighted}
	rt.Params.Algorithm.Solver = sandboxpkg.SolverNSGA2
	rt.Params.Algorithm.SAInitialTemp = 100
	rt.Params.Algorithm.SAItersPerTemp = 10
	objCtx := newObjectiveContext(rt.Params.Objectives, stats)
	out := solveDispatch(rt, objCtx, shipments, vehicles, stats, cargos, cloneProjectedLoads(loads), pois)
	if len(out) != len(shipments) {
		t.Fatalf("weighted 回落 SA 应完成全部指派, got %d", len(out))
	}
}

// ---- 算法部件单测 ----

func TestFastNondominatedSortAndCrowding(t *testing.T) {
	// 3 层结构：A 支配 B/C；B/C 互不支配；D 被 B 支配
	mkEval := func(c2, u2 float64, viol float64) stateEvaluation {
		return stateEvaluation{
			Vector:    ObjectiveVector{C2UnitCost: c2, U2LoadingRate: u2},
			Violation: viol,
		}
	}
	pop := []moeaIndividual{
		{eval: mkEval(1, 0.8, 0)},  // A：最优
		{eval: mkEval(2, 0.5, 0)},  // B
		{eval: mkEval(1.5, 0.7, 0)}, // C
		{eval: mkEval(3, 0.4, 0)},  // D：被 B 支配（C2 3>2 且 U2 0.4<0.5）
	}
	for i := range pop {
		pop[i].obj = minimizeObjectives(pop[i].eval.Vector)
	}
	fronts := fastNondominatedSort(pop)
	if len(fronts) < 2 {
		t.Fatalf("应至少 2 层, got %d", len(fronts))
	}
	rankOf := map[int]int{}
	for r, f := range fronts {
		for _, i := range f {
			rankOf[i] = r + 1
		}
	}
	if rankOf[0] != 1 {
		t.Fatalf("A 应为第 1 层, got %d", rankOf[0])
	}
	if rankOf[3] <= rankOf[1] || rankOf[3] <= rankOf[2] {
		t.Fatalf("D 层级应劣于 B/C: D=%d B=%d C=%d", rankOf[3], rankOf[1], rankOf[2])
	}

	// ε-约束支配：违反量大的解层级劣
	pop2 := []moeaIndividual{
		{eval: mkEval(1, 0.8, 0.5)}, // 违反 0.5
		{eval: mkEval(2, 0.5, 0)},   // 可行
	}
	for i := range pop2 {
		pop2[i].obj = minimizeObjectives(pop2[i].eval.Vector)
	}
	fronts2 := fastNondominatedSort(pop2)
	if len(fronts2) != 2 || fronts2[0][0] != 1 {
		t.Fatalf("可行解应排第 1 层（约束优先于目标）, got %+v", fronts2)
	}

	// 拥挤度：单点前1后1 → 边界个体 Inf
	single := []moeaIndividual{
		{obj: [4]float64{0, 0, 0, 0}, norm: [4]float64{0, 0, 0, 0}},
		{obj: [4]float64{0.5, 0.5, 0.5, 0.5}, norm: [4]float64{0.5, 0.5, 0.5, 0.5}},
		{obj: [4]float64{1, 1, 1, 1}, norm: [4]float64{1, 1, 1, 1}},
	}
	crowdingDistance(single, []int{0, 1, 2})
	if !isInf(single[0].crowd) || !isInf(single[2].crowd) {
		t.Fatalf("边界个体拥挤度应为 Inf, got %v %v", single[0].crowd, single[2].crowd)
	}
}

func isInf(v float64) bool {
	return v > 1e308
}

func TestDasDennisPointsAndLinearSolve(t *testing.T) {
	// H=4, M=4 → C(7,3)=35 个参考点
	pts := dasDennisPoints(4, 4)
	if len(pts) != 35 {
		t.Fatalf("Das-Dennis(4,4) 应 35 点, got %d", len(pts))
	}
	for _, p := range pts {
		sum := 0.0
		for _, x := range p {
			if x < 0 || x > 1 {
				t.Fatalf("参考点分量越界: %v", p)
			}
			sum += x
		}
		if sum < 0.999 || sum > 1.001 {
			t.Fatalf("参考点分量和应为 1: %v", p)
		}
	}
	// H=2, M=2 → C(3,1)=3
	if len(dasDennisPoints(2, 2)) != 3 {
		t.Fatalf("Das-Dennis(2,2) 应 3 点")
	}

	// 4x4 线性求解：单位阵 → x=1
	A := [][4]float64{
		{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 1, 0}, {0, 0, 0, 1},
	}
	x, ok := solveLinear4(A)
	if !ok || x[0] != 1 || x[3] != 1 {
		t.Fatalf("单位阵解应为 1, got %v ok=%v", x, ok)
	}
	// 奇异阵 → 不 ok
	B := [][4]float64{
		{1, 1, 1, 1}, {1, 1, 1, 1}, {0, 0, 1, 0}, {0, 0, 0, 1},
	}
	if _, ok := solveLinear4(B); ok {
		t.Fatal("奇异阵应不可解")
	}
}

func TestMOEADAggregateAndNeighborhood(t *testing.T) {
	ideal := [4]float64{0, 0, 0, 0}
	nadir := [4]float64{1, 1, 1, 1}
	lambda := []float64{0.25, 0.25, 0.25, 0.25}

	// 可行解（各维 0.5）三种聚合均为有限值且违反量惩罚生效
	ind := &moeaIndividual{obj: [4]float64{0.5, -0.5, 0.5, 0.5}, eval: stateEvaluation{Violation: 0.2}}
	for _, decomp := range []string{"tchebycheff", "weighted_sum", "pbi"} {
		v := moeadAggregate(ind, ideal, nadir, lambda, decomp)
		if v <= 0 || v >= SAPenaltyInf {
			t.Fatalf("%s 聚合值异常: %v", decomp, v)
		}
	}
	// 违反量加罚：violation 0.2 → 分数增加 8*0.2
	indOK := &moeaIndividual{obj: ind.obj, eval: stateEvaluation{Violation: 0}}
	base := moeadAggregate(indOK, ideal, nadir, lambda, "tchebycheff")
	withV := moeadAggregate(ind, ideal, nadir, lambda, "tchebycheff")
	if withV-base < softPenaltyWeight*0.19 {
		t.Fatalf("软约束惩罚应生效: base=%v with=%v", base, withV)
	}
	// 不可行一票否决
	bad := &moeaIndividual{obj: ind.obj, eval: stateEvaluation{Infeasible: 1}}
	if moeadAggregate(bad, ideal, nadir, lambda, "tchebycheff") < SAPenaltyInf {
		t.Fatal("不可行解聚合值应含 SAPenaltyInf")
	}

	// 邻域：全连接权重下 T=2 → 每子问题 2 邻居
	lambdas := [][]float64{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 1, 0}}
	B := moeadNeighborhoods(lambdas, 2)
	if len(B) != 3 || len(B[0]) != 2 {
		t.Fatalf("邻域构造异常: %+v", B)
	}
	if B[0][0] != 0 {
		t.Fatalf("自身应在邻域首位（距离 0 最小）, got %v", B[0])
	}
}

func TestSPEA2Fitness(t *testing.T) {
	mkEval := func(c2 float64) stateEvaluation {
		return stateEvaluation{Vector: ObjectiveVector{C2UnitCost: c2, U2LoadingRate: 0.5}}
	}
	combined := []moeaIndividual{
		{eval: mkEval(1)},   // A：支配 B
		{eval: mkEval(2)},   // B：被 A 支配
		{eval: mkEval(1.5)}, // C：被 A 支配
	}
	for i := range combined {
		combined[i].obj = minimizeObjectives(combined[i].eval.Vector)
	}
	moeaNormalize(combined)
	entries := spea2Fitness(combined)
	// A 非支配：raw=0 → fitness<1；B/C raw = S(A)=2 → fitness ≥ 2
	if entries[0].raw != 0 {
		t.Fatalf("非支配个体 raw 应为 0, got %v", entries[0].raw)
	}
	if entries[1].raw != 3 || entries[2].raw != 2 {
		t.Fatalf("被支配个体 raw 应为 S(A)+S(C)=3 与 S(A)=2, got %v %v", entries[1].raw, entries[2].raw)
	}
	if entries[0].fitness >= 1 {
		t.Fatalf("非支配个体 fitness 应 <1, got %v", entries[0].fitness)
	}
	if entries[1].fitness < 2 {
		t.Fatalf("被支配个体 fitness 应 ≥2, got %v", entries[1].fitness)
	}
}

func TestMOPSOArchiveUpdate(t *testing.T) {
	mk := func(c2, u2 float64) moeaIndividual {
		ev := stateEvaluation{Vector: ObjectiveVector{C2UnitCost: c2, U2LoadingRate: u2}}
		return moeaIndividual{assigned: []int{0}, eval: ev, obj: minimizeObjectives(ev.Vector)}
	}
	var archive []moeaIndividual
	archive = mopsoArchiveUpdate(archive, []int{0}, mk(2, 0.5).eval, 10)
	archive = mopsoArchiveUpdate(archive, []int{1}, mk(1, 0.6).eval, 10) // 支配第 1 个
	if len(archive) != 1 {
		t.Fatalf("被支配成员应被剔除, size=%d", len(archive))
	}
	dominated := stateEvaluation{Vector: ObjectiveVector{C2UnitCost: 3, U2LoadingRate: 0.4}}
	archive = mopsoArchiveUpdate(archive, []int{2}, dominated, 10)
	if len(archive) != 1 {
		t.Fatalf("被支配解不应入档, size=%d", len(archive))
	}
	// 容量淘汰：加入互不支配解至超容量
	for k := 0; k < 5; k++ {
		ev := stateEvaluation{Vector: ObjectiveVector{C2UnitCost: float64(1 + k), U2LoadingRate: 0.9}}
		archive = mopsoArchiveUpdate(archive, []int{k}, ev, 3)
	}
	if len(archive) > 3 {
		t.Fatalf("档案容量应 ≤3, got %d", len(archive))
	}
}

// ---- 参数层：solver 字段与超参解析 ----

func TestAlgorithmParamsSolverResolution(t *testing.T) {
	// solver 显式指定
	p, err := sandboxpkg.ParseExperimentParams("[algorithm]\nsolver = \"nsga2\"\npopulation = 20\nref_divisions = 5\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if p.Algorithm.EffectiveSolver() != sandboxpkg.SolverNSGA2 {
		t.Fatalf("solver 解析失败: %s", p.Algorithm.Solver)
	}
	if p.Algorithm.Population != 20 || p.Algorithm.RefDivisions != 5 {
		t.Fatalf("超参解析失败: pop=%d ref=%d", p.Algorithm.Population, p.Algorithm.RefDivisions)
	}
	// 兼容：mode=nf 未指定 solver
	p2, _ := sandboxpkg.ParseExperimentParams("[algorithm]\nmode = \"nf\"\n")
	if p2.Algorithm.EffectiveSolver() != sandboxpkg.SolverNF {
		t.Fatalf("mode=nf 应映射 NF 求解器, got %s", p2.Algorithm.EffectiveSolver())
	}
	// 缺省：回落 SA
	p3, _ := sandboxpkg.ParseExperimentParams("")
	if p3.Algorithm.EffectiveSolver() != sandboxpkg.SolverSA {
		t.Fatalf("缺省应回落 SA, got %s", p3.Algorithm.EffectiveSolver())
	}
	// 默认参数包含种群超参初值
	if sandboxpkg.DefaultExperimentParams().Algorithm.Population != 40 ||
		sandboxpkg.DefaultExperimentParams().Algorithm.Generations != 60 {
		t.Fatal("默认种群超参缺失")
	}
}
