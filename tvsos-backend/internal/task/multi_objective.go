package task

// ============================================================================
// 多目标优化（主要目标法 + 帕累托考量）——指标体系文档 §2.3/§2.5 的求解层落地。
//
// 问题结构：4 个主目标 C2↓(单位成本) / U2↑(装载率) / T2↓(响应P95) / U4↓(空驶率)
// 互为矛盾（U2↔T2、U4↔T2 两组典型权衡），另有 F1 基尼 / G2 碳排两项软约束。
//
// 求解策略（主要目标法 / ε-约束法 + 帕累托考量）：
//  1. 主目标四选一（primary），作为唯一硬性优化对象；
//  2. 其余指标转软约束：绝对预算（budgets）或"相对当前值劣化容差"（tolerance）
//     的自适应 ε 预算；违反量按相对比例折算，进标量分数的惩罚项（软性限制，
//     不可行不硬排除，给主目标留出让步空间）；
//  3. SA 接受准则叠加帕累托支配判定：候选解在 4 主目标上支配当前解时无条件
//     接受（沿帕累托前沿方向自由移动），其余情况按标量化分数走 Metropolis；
//  4. 全程维护有界帕累托档案（非支配解集），终选按标量化分数（= 主目标 +
//     惩罚×违反量，天然"可行优先、主目标次之"的字典序近似）取 ε-约束解。
//
// 评价口径：所有指标均为"批次投影"——SA 在一个派单批次内能直接影响的增量口径
// （C2=批次成本/批次吨公里、U2=指派后装载率均值、U4=批次路线空驶占比、
// T2=批次响应P95 投影），与全局同名指标同源可比但为批次增量，供批次内寻优。
// ============================================================================

import (
	"math"
	"sort"
	"sync"

	"github.com/kiritosuki/mover/internal/model"
	sandboxpkg "github.com/kiritosuki/mover/internal/sandbox"
)

// 主目标代号（与 sandbox.ObjectiveParams 常量一致）。
const (
	objKeyC2 = "C2"
	objKeyU2 = "U2"
	objKeyU4 = "U4"
	objKeyT2 = "T2"
	objKeyF1 = "F1"
	objKeyG2 = "G2"
)

// softPenaltyWeight 软约束惩罚系数：每 1 单位相对违反量（如超预算 100%）
// 折算 8.0 分主目标归一项。取值保证"可满足约束的解近似字典序优先于违反解"。
const softPenaltyWeight = 8.0

// paretoArchiveCap 帕累托档案容量上限（有界，防极端批次内存膨胀）。
const paretoArchiveCap = 32

// primaryScaleFloors 主目标归一化尺度的下限保护（初始解取值过小/为零时兜底）。
var primaryScaleFloors = map[string]float64{
	objKeyC2: 1.0,
	objKeyU2: 0.02,
	objKeyU4: 0.02,
	objKeyT2: 30.0, // 秒
}

// ObjectiveVector 一个候选指派方案在 6 项指标上的批次投影值。
// 4 主目标参与帕累托支配与主要目标法；F1/G2 仅作软约束。
type ObjectiveVector struct {
	C2UnitCost    float64 `json:"c2_unit_cost"`     // C2：批次成本/批次吨公里(吨·米)
	U2LoadingRate float64 `json:"u2_loading_rate"`  // U2：指派后全车队装载率均值
	U4EmptyRate   float64 `json:"u4_empty_rate"`    // U4：批次路线空驶里程/总里程
	T2ResponseP95 float64 `json:"t2_response_p95"`  // T2：批次响应时间投影 P95(秒)
	F1Gini        float64 `json:"f1_gini"`          // F1：指派后装载率分布基尼
	G2CarbonRate  float64 `json:"g2_carbon_rate"`   // G2：批次吨公里碳排放投影
	TonDistance   float64 `json:"ton_distance"`     // 批次吨·米（报告用，与指派无关）
}

// metricValue 按指标代号取向量分量。
func (v ObjectiveVector) metricValue(key string) (float64, bool) {
	switch key {
	case objKeyC2:
		return v.C2UnitCost, true
	case objKeyU2:
		return v.U2LoadingRate, true
	case objKeyU4:
		return v.U4EmptyRate, true
	case objKeyT2:
		return v.T2ResponseP95, true
	case objKeyF1:
		return v.F1Gini, true
	case objKeyG2:
		return v.G2CarbonRate, true
	}
	return 0, false
}

// objectiveDirection 指标优化方向：true=越大越优(U2)，false=越小越优。
func objectiveDirection(key string) bool {
	return key == objKeyU2
}

// stateEvaluation 一个候选方案的完整评估：原始成本口径 + 多目标标量化分数。
type stateEvaluation struct {
	RawCost   float64        // 兼容口径：ΣCostF + LIFO项 + 可行性/绕路惩罚（weighted 模式即评分）
	CleanCost float64        // 纯业务成本：ΣCostF + LIFO 项（不含 1e9/1e7 惩罚项）
	Score     float64        // 多目标标量化分数（weighted 模式 = RawCost）
	Vector    ObjectiveVector
	Violation float64        // 软约束相对违反量总和（≥0）
	Infeasible int           // 不可行（超载/类型不符/LIFO无合法序）车辆数
	DetourHits int           // 绕路超限惩罚命中数
}

// scoreOrRawCost weighted 模式返回原始成本，primary 模式返回多目标分数。
func (e stateEvaluation) scoreOrRawCost(weightedMode bool) float64 {
	if weightedMode {
		return e.RawCost
	}
	return e.Score
}

// objectiveContext 批次级多目标评估上下文：配置解析 + 自适应 ε 预算 +
// 主目标归一化尺度（由 SA 初始解标定一次，全批次共用，保证 M/C1/A3 与
// SA 在同一目标口径下评估）。
type objectiveContext struct {
	weighted bool   // true=旧标量化模式（Cost F 唯一目标）
	primary  string // 主目标代号（primary 模式有效）

	softBudgets   map[string]float64 // 激活的软约束预算（绝对+自适应合并；主目标自身不约束）
	primaryScale  float64            // 主目标归一化尺度
	calibrated    bool
}

// budgetKeyToMetric TOML 预算键 → 指标代号映射（含旧 soft_constraints 键兼容）。
var budgetKeyToMetric = map[string]string{
	"C2_max":           objKeyC2,
	"U2_min":           objKeyU2,
	"U4_max":           objKeyU4,
	"T2_p95_max":       objKeyT2,
	"F1_gini_max":      objKeyF1,
	"G2_max":           objKeyG2,
	"G2_carbon_max":    objKeyG2, // 兼容旧键名
}

// metricFromBudgetKey 预算键查表（无匹配返回空）。
func metricFromBudgetKey(key string) string {
	return budgetKeyToMetric[key]
}

// newObjectiveContext 从实验参数与统计快照构建批次多目标上下文。
// stats 可为 nil（统计快照获取失败/NF 回退）：自适应预算全部失效，仅绝对预算生效。
func newObjectiveContext(params sandboxpkg.ObjectiveParams, stats *Statistics) *objectiveContext {
	mode := params.NormalizedMode()
	primary := params.NormalizedPrimary()
	ctx := &objectiveContext{
		weighted:    mode == sandboxpkg.ObjectiveModeWeighted,
		primary:     primary,
		softBudgets: make(map[string]float64, 6),
	}
	if ctx.weighted {
		return ctx
	}

	// 绝对预算（budgets + 兼容 soft_constraints 合并；主目标自身不约束）
	for k, v := range params.EffectiveBudgets() {
		metric := metricFromBudgetKey(k)
		if metric == "" || metric == primary {
			continue
		}
		ctx.softBudgets[metric] = v
	}

	// 自适应 ε 预算：非主目标三指标锚定当前全局值 × (1±容差)
	tol := params.Tolerance
	if tol < 0 {
		tol = 0
	}
	if tol > 1 {
		tol = 1
	}
	if stats != nil {
		anchors := map[string]float64{
			objKeyC2: stats.UnitCost,
			objKeyU2: stats.UtilizationStats.LoadingRateAvg,
			objKeyU4: stats.EmptyDistStats.EmptyRate,
			objKeyT2: stats.MatchResponseStats.P95,
		}
		for key, anchor := range anchors {
			if key == primary {
				continue
			}
			if _, has := ctx.softBudgets[key]; has {
				continue // 绝对预算优先，不叠加自适应
			}
			if anchor <= 0 {
				continue // 实验初期指标尚无有效值，无法锚定（约束自然失活）
			}
			if objectiveDirection(key) {
				budget := anchor * (1 - tol)
				if budget < 0 {
					budget = 0
				}
				ctx.softBudgets[key] = budget
			} else {
				ctx.softBudgets[key] = anchor * (1 + tol)
			}
		}
	}
	return ctx
}

// calibratePrimaryScale 用 SA 初始解的指标值标定主目标归一化尺度（每批一次）。
func (ctx *objectiveContext) calibratePrimaryScale(v ObjectiveVector) {
	if ctx.weighted || ctx.calibrated {
		return
	}
	raw, _ := v.metricValue(ctx.primary)
	if math.IsNaN(raw) || math.IsInf(raw, 0) || raw < 0 {
		raw = 0
	}
	scale := math.Abs(raw)
	if floor, ok := primaryScaleFloors[ctx.primary]; ok && scale < floor {
		scale = floor
	}
	ctx.primaryScale = scale
	ctx.calibrated = true
}

// primaryTerm 主目标归一项：初始解≈尺度基准（≈1 或 0），越优越小。
func (ctx *objectiveContext) primaryTerm(v ObjectiveVector) float64 {
	raw, ok := v.metricValue(ctx.primary)
	if !ok || math.IsNaN(raw) || math.IsInf(raw, 0) {
		return math.MaxFloat64 / 4 // 异常值按极差处理
	}
	scale := ctx.primaryScale
	if floor, ok := primaryScaleFloors[ctx.primary]; ok && scale < floor {
		scale = floor
	}
	if scale <= 0 {
		scale = 1
	}
	if ctx.primary == objKeyU2 {
		// 最大化目标：折算为 (基准-值)/尺度，越优于越小于 0
		return (scale - raw) / scale
	}
	return raw / scale
}

// violationOf 计算软约束相对违反量总和：每项 = 相对预算的超限比例。
// 预算 > 0 才生效；违反量非负，可满足时为 0。
func (ctx *objectiveContext) violationOf(v ObjectiveVector) float64 {
	total := 0.0
	for key, budget := range ctx.softBudgets {
		if budget <= 0 {
			continue
		}
		val, ok := v.metricValue(key)
		if !ok || math.IsNaN(val) || math.IsInf(val, 0) {
			total += 1 // 异常值按全额违反
			continue
		}
		if objectiveDirection(key) {
			if val < budget { // 下限约束
				total += (budget - val) / budget
			}
		} else {
			if val > budget { // 上限约束
				total += (val - budget) / budget
			}
		}
	}
	return total
}

// score 多目标标量化分数（主要目标法 ε-约束形态）：
// score = 主目标归一项 + 惩罚系数 × 软约束相对违反量 + 硬约束惩罚（不可行/绕路）。
// 硬可行性惩罚与旧口径同量级（SAPenaltyInf / SAPenaltyInf/100），保证任何主目标
// 选择下超载/无解/绕路方案都被一票否决。
func (ctx *objectiveContext) score(cleanCost float64, v ObjectiveVector, infeasible, detourHits int) float64 {
	score := ctx.primaryTerm(v) + softPenaltyWeight*ctx.violationOf(v)
	score += float64(infeasible) * SAPenaltyInf
	score += float64(detourHits) * (SAPenaltyInf / 100.0)
	return score
}

// ---- 帕累托支配（4 主目标空间） ----

// paretoDominance 比较 a/b 在 4 主目标上的帕累托关系。
// 返回 +1：a 支配 b；-1：a 被 b 支配；0：互不支配或相等。
// 支配 = 在所有 4 维上不劣且至少一维严格更优（U2 取越大越优方向）。
func paretoDominance(a, b ObjectiveVector) int {
	better, worse := 0, 0
	compare := func(av, bv float64, maximize bool) {
		if av == bv {
			return
		}
		aBetter := av < bv
		if maximize {
			aBetter = av > bv
		}
		if aBetter {
			better++
		} else {
			worse++
		}
	}
	compare(a.C2UnitCost, b.C2UnitCost, false)
	compare(a.U2LoadingRate, b.U2LoadingRate, true)
	compare(a.U4EmptyRate, b.U4EmptyRate, false)
	compare(a.T2ResponseP95, b.T2ResponseP95, false)
	if better > 0 && worse == 0 {
		return 1
	}
	if worse > 0 && better == 0 {
		return -1
	}
	return 0
}

// ---- 帕累托档案（有界非支配解集） ----

// paretoEntry 档案成员：完整指派方案 + 评估快照。
type paretoEntry struct {
	assigned  []int
	vector    ObjectiveVector
	score     float64
	violation float64
}

// paretoArchive 批次内有界帕累托档案。
// offer 语义：新解不被任何成员支配才入档，同时剔除被其支配的成员；
// 超容量时淘汰"主目标原始值最差"的成员（保前沿优良端，供 ε-终选与报告）。
type paretoArchive struct {
	entries  []paretoEntry
	capacity int
	primary  string
}

func newParetoArchive(primary string) *paretoArchive {
	return &paretoArchive{capacity: paretoArchiveCap, primary: primary}
}

func (a *paretoArchive) size() int { return len(a.entries) }

// offer 提交一个候选解（不修改入参 assigned，内部拷贝）。
// 不可行解（超载/无合法 LIFO 序）的投影向量为空值无意义，不入档案；
// 与既有成员向量完全相同的解不重复入档（不同指派的同值投影，保持前沿多样性）。
func (a *paretoArchive) offer(assigned []int, eval stateEvaluation) {
	if eval.Infeasible > 0 {
		return
	}
	for _, e := range a.entries {
		if e.vector == eval.Vector {
			return
		}
		if paretoDominance(e.vector, eval.Vector) > 0 {
			return // 已被档案成员支配
		}
	}
	kept := a.entries[:0]
	for _, e := range a.entries {
		if paretoDominance(eval.Vector, e.vector) > 0 {
			continue // 剔除被新解支配的成员
		}
		kept = append(kept, e)
	}
	entry := paretoEntry{
		assigned:  append([]int(nil), assigned...),
		vector:    eval.Vector,
		score:     eval.Score,
		violation: eval.Violation,
	}
	kept = append(kept, entry)
	a.entries = kept

	// 容量淘汰：按主目标原始值最差者出档（最大化目标取最小值者）
	if len(a.entries) > a.capacity {
		worstIdx := 0
		worstVal, _ := a.entries[0].vector.metricValue(a.primary)
		for i := 1; i < len(a.entries); i++ {
			val, _ := a.entries[i].vector.metricValue(a.primary)
			if objectiveDirection(a.primary) {
				if val < worstVal {
					worstVal, worstIdx = val, i
				}
			} else {
				if val > worstVal {
					worstVal, worstIdx = val, i
				}
			}
		}
		a.entries = append(a.entries[:worstIdx], a.entries[worstIdx+1:]...)
	}
}

// bestByScore 档案中标量化分数最优的成员（ε-约束语义下的终选候选）。
func (a *paretoArchive) bestByScore() (paretoEntry, bool) {
	if len(a.entries) == 0 {
		return paretoEntry{}, false
	}
	best := a.entries[0]
	for _, e := range a.entries[1:] {
		if e.score < best.score {
			best = e
		}
	}
	return best, true
}

// frontVectors 导出档案前沿的目标向量（帕累托前沿报告口径，按主目标排序）。
func (a *paretoArchive) frontVectors() []ObjectiveVector {
	out := make([]ObjectiveVector, len(a.entries))
	for i, e := range a.entries {
		out[i] = e.vector
	}
	sort.SliceStable(out, func(i, j int) bool {
		vi, _ := out[i].metricValue(a.primary)
		vj, _ := out[j].metricValue(a.primary)
		if objectiveDirection(a.primary) {
			return vi > vj
		}
		return vi < vj
	})
	return out
}

// ---- 实验级多目标运行统计（实时看板 + 终态结果数据源） ----

var (
	objectiveRunMu sync.Mutex
	objectiveRun   = struct {
		batches          int     // 已求解批次数
		feasibleBatches  int     // 终选解软约束可满足（violation==0）的批次数
		violationSum     float64 // 终选解相对违反量累计（均值=violationSum/batches）
		frontSizeSum     int     // 帕累托档案规模累计（均值=frontSizeSum/batches）
		frontSizeMax     int     // 单批档案规模峰值
		primaryKey       string  // 主目标代号（最近批次）
		mode             string  // 优化模式（最近批次）
		lastFront        []ObjectiveVector // 最近一批的帕累托前沿
	}{}
)

// ResetObjectiveRunStats 清空多目标运行统计（实验启动时调用）。
func ResetObjectiveRunStats() {
	objectiveRunMu.Lock()
	objectiveRun.batches, objectiveRun.feasibleBatches = 0, 0
	objectiveRun.violationSum, objectiveRun.frontSizeSum, objectiveRun.frontSizeMax = 0, 0, 0
	objectiveRun.primaryKey, objectiveRun.mode = "", ""
	objectiveRun.lastFront = nil
	objectiveRunMu.Unlock()
}

// recordObjectiveBatch 记录一个批次的求解结果（SA 终选后调用）。
func recordObjectiveBatch(ctx *objectiveContext, chosen stateEvaluation, frontSize int, front []ObjectiveVector) {
	objectiveRunMu.Lock()
	defer objectiveRunMu.Unlock()
	objectiveRun.batches++
	if chosen.Violation <= 0 {
		objectiveRun.feasibleBatches++
	}
	objectiveRun.violationSum += chosen.Violation
	objectiveRun.frontSizeSum += frontSize
	if frontSize > objectiveRun.frontSizeMax {
		objectiveRun.frontSizeMax = frontSize
	}
	if ctx != nil {
		if ctx.weighted {
			objectiveRun.mode = sandboxpkg.ObjectiveModeWeighted
			objectiveRun.primaryKey = "" // weighted 模式无主目标概念
		} else {
			objectiveRun.mode = sandboxpkg.ObjectiveModePrimary
			objectiveRun.primaryKey = ctx.primary
		}
	}
	if len(front) > 0 {
		objectiveRun.lastFront = append([]ObjectiveVector(nil), front...)
	}
}

// ObjectiveRunStatsView 多目标运行统计的对外视图（实时看板/结果层用）。
type ObjectiveRunStatsView struct {
	Mode            string           `json:"mode"`              // primary / weighted
	Primary         string           `json:"primary"`           // 主目标代号
	Batches         int              `json:"batches"`           // 已求解批次数
	FeasibleBatches int              `json:"feasible_batches"`  // 软约束可满足批次数
	FeasiblePct     float64          `json:"feasible_pct"`      // 可满足批次占比(0-100)
	ViolationAvg    float64          `json:"violation_avg"`     // 平均相对违反量
	FrontAvg        float64          `json:"front_avg"`         // 平均帕累托前沿规模
	FrontMax        int              `json:"front_max"`         // 前沿规模峰值
	LastFront       []ObjectiveVector `json:"last_front"`       // 最近批次帕累托前沿
}

// GetObjectiveRunStats 返回多目标运行统计快照。
func GetObjectiveRunStats() ObjectiveRunStatsView {
	objectiveRunMu.Lock()
	defer objectiveRunMu.Unlock()
	view := ObjectiveRunStatsView{
		Mode:        objectiveRun.mode,
		Primary:     objectiveRun.primaryKey,
		Batches:     objectiveRun.batches,
		FrontMax:    objectiveRun.frontSizeMax,
		LastFront:   objectiveRun.lastFront,
	}
	if objectiveRun.batches > 0 {
		view.FeasibleBatches = objectiveRun.feasibleBatches
		view.FeasiblePct = float64(objectiveRun.feasibleBatches) / float64(objectiveRun.batches) * 100
		view.ViolationAvg = objectiveRun.violationSum / float64(objectiveRun.batches)
		view.FrontAvg = float64(objectiveRun.frontSizeSum) / float64(objectiveRun.batches)
	}
	return view
}

// vehicleObjectiveProjection 单车层面的多目标投影数据。
// 由 evaluateVehicleTasks 在既有成本遍历中顺带产出，避免二次扫描。
type vehicleObjectiveProjection struct {
	RouteDist    float64   // LIFO 最优路线总距离(米)
	LoadedDist   float64   // 载重段距离(米)：Σ 起终点直线距离
	TonDistance  float64   // 批次吨·米：Σ 吨位×起终点直线距离
	LoadKg       int       // 派单后累计载重(含既有预占)
	ResponseSecs []float64 // 各任务响应投影(秒)：现有等待 + 接驾时间
}

// carbonOfKm 按车型排放因子估算碳排放(kg)。空驶/满载因子缺失时回落默认估算。
func carbonOfKm(vehicleType int, emptyKm, loadedKm float64) float64 {
	fEmpty, fLoad := defaultEmissionFactors(vehicleType)
	if emissionFactorProvider != nil {
		if e, l := emissionFactorProvider(vehicleType); e > 0 || l > 0 {
			fEmpty, fLoad = e, l
		}
	}
	return emptyKm*fEmpty + loadedKm*fLoad
}

// buildObjectiveVector 由批次聚合量组装目标向量（与全局指标同源的批次投影口径）。
// 输入为全部参与车辆的投影与全局车队装载率分布。
func buildObjectiveVector(
	vehicles []*model.Vehicle,
	projections map[int]*vehicleObjectiveProjection, // vehicle index -> projection
	loadingRates []float64, // 指派后全车队(含未参与车辆)单车装载率
	cleanCost float64,
) ObjectiveVector {
	v := ObjectiveVector{}

	// 批次吨·米（与指派无关的常量）
	var tonDist, routeDist, loadedDist float64
	var responses []float64
	var carbonKg float64
	for _, p := range projections {
		tonDist += p.TonDistance
		routeDist += p.RouteDist
		loadedDist += p.LoadedDist
		responses = append(responses, p.ResponseSecs...)
	}
	v.TonDistance = tonDist

	// C2：批次成本/吨公里（分母恒定，与全局 C2=F/总吨公里 同构）
	if tonDist > 0 {
		v.C2UnitCost = cleanCost / tonDist
	}

	// U2：指派后装载率均值（全局 U2 同口径：容量>0 车辆的 size/capacity 均值）
	if len(loadingRates) > 0 {
		sum := 0.0
		for _, r := range loadingRates {
			sum += r
		}
		v.U2LoadingRate = sum / float64(len(loadingRates))
	}

	// U4：批次路线空驶占比 = (路线-载重段)/路线
	if routeDist > 0 {
		empty := routeDist - loadedDist
		if empty < 0 {
			empty = 0
		}
		v.U4EmptyRate = empty / routeDist
	}

	// T2：批次响应投影 P95（与 T2 全局口径同为线性插值分位）
	if len(responses) > 0 {
		sort.Float64s(responses)
		v.T2ResponseP95 = computePercentile(responses, 95)
	}

	// F1：指派后装载率分布基尼（与全局 F1 同公式）
	v.F1Gini = giniCoefficient(loadingRates)

	// G2：批次吨公里碳排放投影
	for vi, p := range projections {
		if p.RouteDist <= 0 {
			continue
		}
		emptyKm := (p.RouteDist - p.LoadedDist) / 1000.0
		if emptyKm < 0 {
			emptyKm = 0
		}
		loadedKm := p.LoadedDist / 1000.0
		carbonKg += carbonOfKm(vehicles[vi].Tybe, emptyKm, loadedKm)
	}
	if tonDist > 0 {
		v.G2CarbonRate = carbonKg / tonDist
	}
	return v
}
