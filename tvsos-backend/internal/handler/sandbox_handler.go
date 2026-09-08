package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kiritosuki/mover/internal/model"
	"github.com/kiritosuki/mover/internal/repository"
	"github.com/kiritosuki/mover/internal/result"
	sandboxpkg "github.com/kiritosuki/mover/internal/sandbox"
)

// ============================================================================
// 沙箱系统 API：沙箱 / 实验方案 / 实验(批次) / 数据生成 / 生命周期 / 种子导入导出 / 对比
// ============================================================================

// ---------------- 沙箱 ----------------

type sandboxUpsertRequest struct {
	Name           string `json:"name" binding:"required"`
	Description    string `json:"description"`
	WorldRulesToml string `json:"worldRulesToml"`
	FleetTemplate  string `json:"fleetTemplate"`
	PoiWhitelist   string `json:"poiWhitelist"`
}

// ListSandboxes 沙箱列表
func ListSandboxes(c *gin.Context) {
	sandboxes, err := repository.ListSandboxes(c.Query("archived") == "1")
	if err != nil {
		result.Fail(c, "查询沙箱列表失败", err)
		return
	}
	result.Success(c, sandboxes)
}

// CreateSandbox 新建沙箱（世界规则留空时取默认 TOML）
func CreateSandbox(c *gin.Context) {
	var req sandboxUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, "参数不完整：name 必填", err)
		return
	}
	rulesToml := req.WorldRulesToml
	if rulesToml == "" {
		rules, _ := sandboxpkg.MarshalWorldRules(sandboxpkg.DefaultWorldRules())
		rulesToml = rules
	} else if _, err := sandboxpkg.ParseWorldRules(rulesToml); err != nil {
		result.Fail(c, "世界规则 TOML 解析失败: "+err.Error(), err)
		return
	}
	fleet := req.FleetTemplate
	if fleet == "" {
		fleet = "[]"
	}
	whitelist := req.PoiWhitelist
	if whitelist == "" {
		whitelist = "[]"
	}
	sb := &model.Sandbox{
		Name:           req.Name,
		Description:    req.Description,
		WorldRulesToml: rulesToml,
		FleetTemplate:  fleet,
		PoiWhitelist:   whitelist,
		AlgoVersion:    "sa-v2.1",
		CreatedBy:      c.GetUint("userId"),
	}
	if err := repository.CreateSandbox(sb); err != nil {
		result.Fail(c, "创建沙箱失败", err)
		return
	}
	result.Success(c, sb)
}

// GetSandbox 沙箱详情
func GetSandbox(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		result.Fail(c, "id 非法", err)
		return
	}
	sb, err := repository.GetSandbox(uint(id))
	if err != nil {
		result.Fail(c, "沙箱不存在", err)
		return
	}
	result.Success(c, sb)
}

// UpdateSandbox 更新沙箱（世界规则/车队模板/描述）
func UpdateSandbox(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		result.Fail(c, "id 非法", err)
		return
	}
	var req sandboxUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, "参数不完整", err)
		return
	}
	if req.WorldRulesToml != "" {
		if _, err := sandboxpkg.ParseWorldRules(req.WorldRulesToml); err != nil {
			result.Fail(c, "世界规则 TOML 解析失败: "+err.Error(), err)
			return
		}
	}
	fields := map[string]interface{}{
		"name":           req.Name,
		"description":    req.Description,
		"world_rules_toml": req.WorldRulesToml,
		"fleet_template": req.FleetTemplate,
		"poi_whitelist":  req.PoiWhitelist,
	}
	if err := repository.UpdateSandboxFields(uint(id), fields); err != nil {
		result.Fail(c, "更新沙箱失败", err)
		return
	}
	result.Success(c, gin.H{"id": id})
}

// ArchiveSandbox 归档沙箱（软删除）
func ArchiveSandbox(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		result.Fail(c, "id 非法", err)
		return
	}
	if err := repository.UpdateSandboxFields(uint(id), map[string]interface{}{"archived": true}); err != nil {
		result.Fail(c, "归档失败", err)
		return
	}
	result.Success(c, gin.H{"id": id, "archived": true})
}

// ---------------- 实验方案 ----------------

type schemeUpsertRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	ParamsToml  string `json:"paramsToml"`
}

// ListSchemes 沙箱下方案列表
func ListSchemes(c *gin.Context) {
	sandboxID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		result.Fail(c, "沙箱 id 非法", err)
		return
	}
	schemes, err := repository.ListSchemesBySandbox(uint(sandboxID))
	if err != nil {
		result.Fail(c, "查询方案列表失败", err)
		return
	}
	result.Success(c, schemes)
}

// CreateScheme 新建实验方案（参数模板留空时取默认）
func CreateScheme(c *gin.Context) {
	sandboxID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		result.Fail(c, "沙箱 id 非法", err)
		return
	}
	var req schemeUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, "参数不完整：name 必填", err)
		return
	}
	paramsToml := req.ParamsToml
	if paramsToml == "" {
		paramsToml, _ = sandboxpkg.MarshalExperimentParams(sandboxpkg.DefaultExperimentParams())
	} else if _, err := sandboxpkg.ParseExperimentParams(paramsToml); err != nil {
		result.Fail(c, "实验参数 TOML 解析失败: "+err.Error(), err)
		return
	}
	scheme := &model.ExperimentScheme{
		SandboxId:   uint(sandboxID),
		Name:        req.Name,
		Description: req.Description,
		ParamsToml:  paramsToml,
		CreatedBy:   c.GetUint("userId"),
	}
	if err := repository.CreateScheme(scheme); err != nil {
		result.Fail(c, "创建方案失败", err)
		return
	}
	result.Success(c, scheme)
}

// UpdateScheme 更新实验方案
func UpdateScheme(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("schemeId"))
	if err != nil {
		result.Fail(c, "方案 id 非法", err)
		return
	}
	var req schemeUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, "参数不完整", err)
		return
	}
	if req.ParamsToml != "" {
		if _, err := sandboxpkg.ParseExperimentParams(req.ParamsToml); err != nil {
			result.Fail(c, "实验参数 TOML 解析失败: "+err.Error(), err)
			return
		}
	}
	if err := repository.UpdateSchemeFields(uint(id), map[string]interface{}{
		"name": req.Name, "description": req.Description, "params_toml": req.ParamsToml,
	}); err != nil {
		result.Fail(c, "更新方案失败", err)
		return
	}
	result.Success(c, gin.H{"id": id})
}

// DeleteScheme 删除实验方案
func DeleteScheme(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("schemeId"))
	if err != nil {
		result.Fail(c, "方案 id 非法", err)
		return
	}
	if err := repository.DeleteScheme(uint(id)); err != nil {
		result.Fail(c, "删除方案失败", err)
		return
	}
	result.Success(c, gin.H{"id": id, "deleted": true})
}
