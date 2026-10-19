package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/kiritosuki/mover/internal/model"
	"github.com/kiritosuki/mover/internal/repository"
	"github.com/kiritosuki/mover/internal/result"
	sandboxpkg "github.com/kiritosuki/mover/internal/sandbox"
	"github.com/kiritosuki/mover/internal/sandbox/gen"
	"github.com/kiritosuki/mover/internal/task"
	"go.uber.org/zap"
)

// ============================================================================
// 实验（批次）API：CRUD / 数据生成 / 生命周期 / 种子导入导出 / 结果与对比
// ============================================================================

type experimentCreateRequest struct {
	SchemeId   uint   `json:"schemeId"`   // 来源方案（可空=临时实验）
	Name       string `json:"name"`
	ParamsToml string `json:"paramsToml"` // 可空：取方案模板/默认
	Seed       *int64 `json:"seed"`       // 可空：取参数内 seed 或默认 12345
}

// ListExperiments 沙箱下实验列表
func ListExperiments(c *gin.Context) {
	sandboxID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		result.Fail(c, "沙箱 id 非法", err)
		return
	}
	experiments, err := repository.ListExperimentsBySandbox(uint(sandboxID))
	if err != nil {
		result.Fail(c, "查询实验列表失败", err)
		return
	}
	result.Success(c, experiments)
}

// CreateExperiment 新建实验（批次）：参数快照 + 种子，状态 configured。
func CreateExperiment(c *gin.Context) {
	sandboxID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		result.Fail(c, "沙箱 id 非法", err)
		return
	}
	var req experimentCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil && req.Name == "" {
		// 允许空 body：全部取默认
	}

	paramsToml := req.ParamsToml
	if paramsToml == "" && req.SchemeId > 0 {
		if scheme, err := repository.GetScheme(req.SchemeId); err == nil {
			paramsToml = scheme.ParamsToml
		}
	}
	if paramsToml == "" {
		paramsToml, _ = sandboxpkg.MarshalExperimentParams(sandboxpkg.DefaultExperimentParams())
	}
	params, err := sandboxpkg.ParseExperimentParams(paramsToml)
	if err != nil {
		result.Fail(c, "实验参数 TOML 解析失败: "+err.Error(), err)
		return
	}
	seed := params.Seed
	if req.Seed != nil {
		seed = *req.Seed
		params.Seed = seed
		paramsToml, _ = sandboxpkg.MarshalExperimentParams(params)
	}
	name := req.Name
	if name == "" {
		name = fmt.Sprintf("实验-%s", time.Now().Format("0102-150405"))
	}
	exp := &model.Experiment{
		SandboxId:   uint(sandboxID),
		SchemeId:    req.SchemeId,
		Name:        name,
		ParamsToml:  paramsToml,
		Seed:        seed,
		Status:      model.ExperimentStatusConfigured,
		DataVersion: 0,
		CreatedBy:   c.GetUint("userId"),
	}
	if err := repository.CreateExperiment(exp); err != nil {
		result.Fail(c, "创建实验失败", err)
		return
	}
	result.Success(c, exp)
}

// GetExperiment 实验详情（含参数 TOML 全文与订单/事件数量）
func GetExperiment(c *gin.Context) {
	exp, ok := mustExperiment(c)
	if !ok {
		return
	}
	orderCount, _ := repository.CountExperimentOrders(exp.Id)
	eventCount, _ := repository.CountExperimentEvents(exp.Id)
	result.Success(c, gin.H{
		"experiment": exp,
		"orderCount": orderCount,
		"eventCount": eventCount,
	})
}

// UpdateExperimentParams 更新实验参数（仅 configured 状态可改）
func UpdateExperimentParams(c *gin.Context) {
	exp, ok := mustExperiment(c)
	if !ok {
		return
	}
	if exp.Status != model.ExperimentStatusConfigured {
		result.Fail(c, "仅 configured 状态的实验可修改参数", errors.New("invalid status"))
		return
	}
	var req struct {
		Name       string `json:"name"`
		ParamsToml string `json:"paramsToml" binding:"required"`
		Seed       *int64  `json:"seed"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, "paramsToml 必填", err)
		return
	}
	params, err := sandboxpkg.ParseExperimentParams(req.ParamsToml)
	if err != nil {
		result.Fail(c, "实验参数 TOML 解析失败: "+err.Error(), err)
		return
	}
	seed := params.Seed
	if req.Seed != nil {
		seed = *req.Seed
	}
	fields := map[string]interface{}{"params_toml": req.ParamsToml, "seed": seed}
	if req.Name != "" {
		fields["name"] = req.Name
	}
	if err := repository.UpdateExperimentFields(exp.Id, fields); err != nil {
		result.Fail(c, "更新实验失败", err)
		return
	}
	result.Success(c, gin.H{"id": exp.Id})
}

// GenerateExperimentData 触发数据生成：确定性预生成订单流 + 事件脚本，状态 → ready。
func GenerateExperimentData(c *gin.Context) {
	exp, ok := mustExperiment(c)
	if !ok {
		return
	}
	if exp.Status != model.ExperimentStatusConfigured && exp.Status != model.ExperimentStatusReady {
		result.Fail(c, "仅 configured/ready 状态的实验可生成数据", errors.New("invalid status: "+exp.Status))
		return
	}
	_ = repository.UpdateExperimentFields(exp.Id, map[string]interface{}{"status": model.ExperimentStatusGenerating})

	orderCount, eventCount, err := generateExperimentData(exp)
	if err != nil {
		_ = repository.UpdateExperimentFields(exp.Id, map[string]interface{}{"status": model.ExperimentStatusFailed})
		result.Fail(c, "数据生成失败: "+err.Error(), err)
		return
	}
	_ = repository.UpdateExperimentFields(exp.Id, map[string]interface{}{
		"status":       model.ExperimentStatusReady,
		"data_version": sandboxpkg.DataGeneratorVersion,
	})
	result.Success(c, gin.H{"orderCount": orderCount, "eventCount": eventCount, "dataVersion": sandboxpkg.DataGeneratorVersion})
}

// generateExperimentData 生成核心：解析世界规则+实验参数 → 调生成器 → 落库（可重入：先清旧脚本）。
func generateExperimentData(exp *model.Experiment) (int, int, error) {
	sb, err := repository.GetSandbox(exp.SandboxId)
	if err != nil {
		return 0, 0, fmt.Errorf("沙箱不存在: %w", err)
	}
	rules, err := sandboxpkg.ParseWorldRules(sb.WorldRulesToml)
	if err != nil {
		return 0, 0, fmt.Errorf("世界规则解析失败: %w", err)
	}
	params, err := sandboxpkg.ParseExperimentParams(exp.ParamsToml)
	if err != nil {
		return 0, 0, fmt.Errorf("实验参数解析失败: %w", err)
	}

	// 车队车牌（与运行时实例化同源：模板前 VehicleCount 辆）
	var template []struct {
		License string `json:"license"`
	}
	if err := json.Unmarshal([]byte(sb.FleetTemplate), &template); err != nil {
		return 0, 0, fmt.Errorf("车队模板解析失败: %w", err)
	}
	count := params.Fleet.VehicleCount
	if count <= 0 || count > len(template) {
		count = len(template)
	}
	licenses := make([]string, 0, count)
	for _, tv := range template[:count] {
		licenses = append(licenses, tv.License)
	}

	// POI 集合（白名单过滤）
	pois, err := repository.ListAllPois()
	if err != nil {
		return 0, 0, err
	}
	whitelist := make(map[uint]bool)
	if len(rules.Geography.PoiWhitelist) > 0 {
		for _, id := range rules.Geography.PoiWhitelist {
			whitelist[id] = true
		}
	}
	poiIDs := make([]uint, 0, len(pois))
	for _, p := range pois {
		if len(whitelist) == 0 || whitelist[p.Id] {
			poiIDs = append(poiIDs, p.Id)
		}
	}
	cargos, err := repository.ListCargos()
	if err != nil {
		return 0, 0, err
	}
	cargoIDs := make([]uint, 0, len(cargos))
	for _, cg := range cargos {
		cargoIDs = append(cargoIDs, cg.Id)
	}

	// 供应链链条解析：货类按名、POI 按类型+名称关键词；任一节点解析失败则整链跳过
	chains := resolveSupplyChains(rules, pois, cargos)

	startTime, err := time.ParseInLocation("2006-01-02 15:04:05", rules.Time.StartTime, time.Local)
	if err != nil {
		startTime = time.Now()
	}
	eventRules := sandboxpkg.EffectiveRandomEvents(params, rules)
	out := gen.Generate(gen.Input{
		ExperimentID:    exp.Id,
		Seed:            exp.Seed,
		DurationHours:   params.Simulation.DurationHours,
		StartTime:       startTime,
		TickSeconds:     rules.Time.TickSeconds,
		Orders:          sandboxpkg.OrderWorldRules{ArrivalModel: rules.Orders.ArrivalModel, DailyVolume: sandboxpkg.EffectiveDailyVolume(params, rules), WeightRangeKg: rules.Orders.WeightRangeKg, CountRange: rules.Orders.CountRange},
		Events:          eventRules,
		VehicleLicenses: licenses,
		PoiIDs:          poiIDs,
		CargoIDs:        cargoIDs,
		Chains:          chains,
	})

	// 重入：清旧脚本后写入
	_ = repository.DeleteExperimentOrdersByExperiment(exp.Id)
	_ = repository.DeleteExperimentEventsByExperiment(exp.Id)
	if err := repository.CreateExperimentOrders(out.Orders); err != nil {
		return 0, 0, err
	}
	if err := repository.CreateExperimentEvents(out.Events); err != nil {
		return 0, 0, err
	}
	// 司机等级表落实验记录（运行时实例化与脚本事件同源）
	levelsJSON, _ := json.Marshal(out.DriverLevels)
	_ = repository.UpdateExperimentFields(exp.Id, map[string]interface{}{"driver_levels": string(levelsJSON)})
	return len(out.Orders), len(out.Events), nil
}

// resolveSupplyChains 将世界规则中的供应链链条解析为生成器输入：
// 节点 POI 按（类型 + 名称关键词，空关键词=仅按类型）在白名单过滤后的 POI 集内解析，
// 货类按名称全局解析；任一节点无 POI 或货类缺失则整链跳过（世界规则与数据不匹配时降级，不报错）。
func resolveSupplyChains(rules sandboxpkg.WorldRules, pois []*model.Poi, cargos []*model.Cargo) []gen.ChainInput {
	if !rules.SupplyChains.Enabled || len(rules.SupplyChains.Chains) == 0 {
		return nil
	}
	cargoByName := make(map[string]*model.Cargo, len(cargos))
	for _, cg := range cargos {
		cargoByName[cg.Name] = cg
	}
	resolved := make([]gen.ChainInput, 0, len(rules.SupplyChains.Chains))
	for _, spec := range rules.SupplyChains.Chains {
		if spec.Share <= 0 || len(spec.Nodes) < 2 {
			continue
		}
		// 逐节点解析 POI 池（发货用自身节点 POI，收货用下一节点 POI）
		nodePois := make([][]uint, len(spec.Nodes))
		valid := true
		for i, node := range spec.Nodes {
			pool := make([]uint, 0)
			for _, p := range pois {
				if p.Tybe != node.PoiTybe {
					continue
				}
				if node.PoiKeyword != "" && !strings.Contains(p.Name, node.PoiKeyword) {
					continue
				}
				pool = append(pool, p.Id)
			}
			if len(pool) == 0 {
				valid = false
				break
			}
			nodePois[i] = pool
		}
		if !valid {
			Logger.Logger.Warn("供应链链条节点无可用POI，整链跳过", zap.String("chain", spec.Name))
			continue
		}
		// 逐段解析货类：步骤k = 节点k的货物发往节点k+1（末节点无货物，不产生运输段）
		steps := make([]gen.ChainStepInput, 0, len(spec.Nodes)-1)
		for k := 0; k < len(spec.Nodes)-1; k++ {
			cargoSpec := spec.Nodes[k]
			cg, ok := cargoByName[cargoSpec.CargoName]
				if !ok || cargoSpec.CargoName == "" {
					Logger.Logger.Warn("供应链链条货类缺失，整链跳过", zap.String("chain", spec.Name), zap.String("cargo", cargoSpec.CargoName))
					valid = false
					break
				}
			trigger := cargoSpec.TriggerProb
			if trigger <= 0 {
					trigger = 1
			}
			yield := cargoSpec.YieldWeight
			if yield <= 0 {
					yield = 1
			}
			steps = append(steps, gen.ChainStepInput{
				CargoId:     cg.Id,
				CargoWeight: cg.Weight,
				FromPoiIDs:  nodePois[k],
				ToPoiIDs:    nodePois[k+1],
				YieldWeight: yield,
				TriggerProb: trigger,
				DelaySec:    int(cargoSpec.DelayMinutes * 60),
				NodeLabel:   cargoSpec.Label,
			})
		}
		if !valid || len(steps) == 0 {
			continue
		}
		resolved = append(resolved, gen.ChainInput{Name: spec.Name, Share: spec.Share, Steps: steps})
	}
	return resolved
}

// ---------------- 生命周期 ----------------

// StartExperiment 启动实验（单实验串行，要求 ready）
func StartExperiment(c *gin.Context) {
	exp, ok := mustExperiment(c)
	if !ok {
		return
	}
	if err := task.StartExperiment(exp.Id); err != nil {
		result.Fail(c, "启动失败: "+err.Error(), err)
		return
	}
	result.Success(c, gin.H{"id": exp.Id, "status": model.ExperimentStatusRunning})
}

// StopExperiment 人工停止实验（状态 aborted，不写终态结果）
func StopExperiment(c *gin.Context) {
	if _, ok := mustExperiment(c); !ok {
		return
	}
	if err := task.StopExperiment(false); err != nil {
		result.Fail(c, "停止失败: "+err.Error(), err)
		return
	}
	result.Success(c, gin.H{"status": model.ExperimentStatusAborted})
}

// GetExperimentStatus 实验运行状态（进度/虚拟时钟/倍速）
func GetExperimentStatus(c *gin.Context) {
	exp, ok := mustExperiment(c)
	if !ok {
		return
	}
	resp := gin.H{
		"id":       exp.Id,
		"status":   exp.Status,
		"progress": exp.Progress,
	}
	if rt := task.CurrentRuntime(); rt != nil && rt.Exp.Id == exp.Id {
		resp["virtualTime"] = rt.Clock.Now().Format("2006-01-02 15:04:05")
		resp["speed"] = rt.GetSpeed()
		resp["progress"] = rt.Clock.Progress()
	}
	result.Success(c, resp)
}

// ---------------- 种子导入导出（可复现性） ----------------

// ExportExperiment 导出实验种子包（参数 TOML + 种子 + 生成器版本），一键复现入口。
func ExportExperiment(c *gin.Context) {
	exp, ok := mustExperiment(c)
	if !ok {
		return
	}
	result.Success(c, gin.H{
		"name":        exp.Name,
		"seed":        exp.Seed,
		"dataVersion": exp.DataVersion,
		"paramsToml":  exp.ParamsToml,
		"sandboxName": mustSandboxName(exp.SandboxId),
		"exportedAt":  time.Now().Format("2006-01-02 15:04:05"),
	})
}

// ImportExperiment 导入种子包：在指定沙箱下创建新实验（configured，需重新 generate 后运行）。
func ImportExperiment(c *gin.Context) {
	sandboxID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		result.Fail(c, "沙箱 id 非法", err)
		return
	}
	var req struct {
		Name        string `json:"name"`
		Seed        int64  `json:"seed" binding:"required"`
		DataVersion int    `json:"dataVersion"`
		ParamsToml  string `json:"paramsToml" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, "种子包不完整：seed 与 paramsToml 必填", err)
		return
	}
	if req.DataVersion > sandboxpkg.DataGeneratorVersion {
		result.Fail(c, fmt.Sprintf("种子包生成器版本 v%d 高于当前 v%d，无法保证可复现", req.DataVersion, sandboxpkg.DataGeneratorVersion), errors.New("incompatible data version"))
		return
	}
	if _, err := sandboxpkg.ParseExperimentParams(req.ParamsToml); err != nil {
		result.Fail(c, "实验参数 TOML 解析失败: "+err.Error(), err)
		return
	}
	name := req.Name
	if name == "" {
		name = fmt.Sprintf("导入实验-%s", time.Now().Format("0102-150405"))
	}
	exp := &model.Experiment{
		SandboxId:  uint(sandboxID),
		Name:       name,
		ParamsToml: req.ParamsToml,
		Seed:       req.Seed,
		Status:     model.ExperimentStatusConfigured,
		CreatedBy:  c.GetUint("userId"),
	}
	if err := repository.CreateExperiment(exp); err != nil {
		result.Fail(c, "导入失败", err)
		return
	}
	result.Success(c, exp)
}

// ---------------- 结果与对比 ----------------

// GetExperimentResult 实验终态结果 + 指标时间序列（结果集查看）
func GetExperimentResult(c *gin.Context) {
	exp, ok := mustExperiment(c)
	if !ok {
		return
	}
	res, err := repository.GetResultByExperiment(exp.Id)
	if err != nil {
		result.Fail(c, "该实验尚无终态结果（可能未完成或已中止）", err)
		return
	}
	snapshots, _ := repository.ListMetricSnapshotsByExp(exp.Id, 0)
	result.Success(c, gin.H{
		"experiment": exp,
		"result":     res,
		"snapshots":  snapshots,
	})
}

// CompareExperiments 多实验对比（同沙箱/跨沙箱）：返回实验元数据 + 终态结果主目标。
func CompareExperiments(c *gin.Context) {
	idsStr := c.Query("exp_ids")
	if idsStr == "" {
		result.Fail(c, "exp_ids 必填（逗号分隔）", errors.New("missing exp_ids"))
		return
	}
	ids := make([]uint, 0, 8)
	for _, s := range splitComma(idsStr) {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			ids = append(ids, uint(v))
		}
	}
	if len(ids) == 0 {
		result.Fail(c, "exp_ids 无有效 id", errors.New("no valid id"))
		return
	}
	experiments := make([]*model.Experiment, 0, len(ids))
	for _, id := range ids {
		if exp, err := repository.GetExperiment(id); err == nil {
			experiments = append(experiments, exp)
		}
	}
	results, _ := repository.ListResultsByExperimentIDs(ids)
	result.Success(c, gin.H{"experiments": experiments, "results": results})
}

// ---------------- 工具 ----------------

func mustExperiment(c *gin.Context) (*model.Experiment, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		result.Fail(c, "实验 id 非法", err)
		return nil, false
	}
	exp, err := repository.GetExperiment(uint(id))
	if err != nil {
		result.Fail(c, "实验不存在", err)
		return nil, false
	}
	return exp, true
}

func mustSandboxName(id uint) string {
	if sb, err := repository.GetSandbox(id); err == nil {
		return sb.Name
	}
	return ""
}

func splitComma(s string) []string {
	out := make([]string, 0, 8)
	cur := ""
	for _, ch := range s {
		if ch == ',' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(ch)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
