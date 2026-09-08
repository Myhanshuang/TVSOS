// 一次性迁移脚本：把现有演示数据（车辆/货类）包装为默认"演示沙箱"，并创建种子账号。
// 用法：cd tvsos-backend && go run scripts/migrate_default_sandbox.go
// 幂等：已有 user/sandbox 时跳过对应步骤。
package main

import (
	"encoding/json"
	"fmt"

	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/kiritosuki/mover/internal/config"
	"github.com/kiritosuki/mover/internal/database"
	"github.com/kiritosuki/mover/internal/model"
	"github.com/kiritosuki/mover/internal/repository"
	"github.com/kiritosuki/mover/internal/sandbox"
	"golang.org/x/crypto/bcrypt"
)

// fleetTemplateVehicle 车队模板中的车辆定义（实验启动时按它实例化本实验专属 vehicle 行）
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

func main() {
	Logger.InitLogger()
	config.InitViper()
	database.InitDB()

	// 1. 种子账号 admin/admin123
	userCount, err := repository.CountUsers()
	if err != nil {
		panic(err)
	}
	if userCount == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		if err != nil {
			panic(err)
		}
		admin := &model.User{Username: "admin", PasswordHash: string(hash), DisplayName: "管理员"}
		if err := repository.CreateUser(admin); err != nil {
			panic(err)
		}
		fmt.Println("已创建种子账号 admin/admin123")
	} else {
		fmt.Println("用户已存在，跳过种子账号创建")
	}

	// 2. 默认沙箱（演示沙箱）：世界规则默认 TOML + 车队模板来自存量车辆(experiment_id=0)
	sandboxCount, err := repository.CountSandboxes()
	if err != nil {
		panic(err)
	}
	if sandboxCount > 0 {
		fmt.Println("沙箱已存在，跳过默认沙箱迁移")
		return
	}

	rules := sandbox.DefaultWorldRules()
	rules.Meta.Name = "演示沙箱"
	rulesToml, err := sandbox.MarshalWorldRules(rules)
	if err != nil {
		panic(err)
	}

	vehicles, err := repository.ListVehicles(0)
	if err != nil {
		panic(err)
	}
	fleet := make([]fleetTemplateVehicle, 0, len(vehicles))
	for _, v := range vehicles {
		if v == nil || v.ExperimentId != 0 {
			continue // 仅模板车辆(experiment_id=0)进入车队模板
		}
		fleet = append(fleet, fleetTemplateVehicle{
			License:  v.License,
			Tybe:     v.Tybe,
			Size:     v.Size,
			Capacity: v.Capacity,
			Length:   v.Length,
			Width:    v.Width,
			Height:   v.Height,
			Lon:      v.Lon,
			Lat:      v.Lat,
		})
	}
	fleetJSON, err := json.Marshal(fleet)
	if err != nil {
		panic(err)
	}

	sb := &model.Sandbox{
		Name:           "演示沙箱",
		Description:    "由存量演示数据自动迁移：1146个成都POI(全局共享) + 20辆川A车队模板 + 默认世界规则",
		WorldRulesToml: rulesToml,
		FleetTemplate:  string(fleetJSON),
		PoiWhitelist:   "[]",
		AlgoVersion:    rules.Meta.AlgoVersion,
		CreatedBy:      1,
	}
	if err := repository.CreateSandbox(sb); err != nil {
		panic(err)
	}
	fmt.Printf("已创建默认沙箱 id=%d，车队模板车辆数=%d\n", sb.Id, len(fleet))

	// 3. 默认实验方案（参数模板）
	paramsToml, err := sandbox.MarshalExperimentParams(sandbox.DefaultExperimentParams())
	if err != nil {
		panic(err)
	}
	scheme := &model.ExperimentScheme{
		SandboxId:   sb.Id,
		Name:        "默认方案",
		Description: "默认参数模板：24h虚拟时长，SA默认超参，C2/U2/U4/T2主目标权重",
		ParamsToml:  paramsToml,
		CreatedBy:   1,
	}
	if err := repository.CreateScheme(scheme); err != nil {
		panic(err)
	}
	fmt.Printf("已创建默认实验方案 id=%d\n", scheme.Id)
}
