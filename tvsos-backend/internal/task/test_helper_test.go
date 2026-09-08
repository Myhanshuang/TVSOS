package task

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/kiritosuki/mover/internal/config"
	"github.com/kiritosuki/mover/internal/database"
	"github.com/kiritosuki/mover/internal/model"
	sandboxpkg "github.com/kiritosuki/mover/internal/sandbox"
)

// setupTestDB 测试数据库初始化（安全红线）：
// 1) 强制读取 config-test.yml（不存在则 Skip），绝不使用 config-dev.yml；
// 2) 连接后校验库名必须包含 "test"，防止误连生产库导致 DELETE 清库事故。
// 所有会清库的测试必须先调用本函数。
func setupTestDB(t *testing.T) {
	t.Helper()
	Logger.InitLogger()
	if err := config.InitViperWithName("config-test"); err != nil {
		t.Skipf("跳过：未找到 config-test.yml（测试必须指向独立测试库，严禁使用 config-dev.yml）: %v", err)
	}
	dbName := config.VP.GetString("database.dbname")
	if !strings.Contains(strings.ToLower(dbName), "test") {
		t.Skipf("跳过：config-test.yml 的 database.dbname=%q 不含 test 字样，为避免清库事故拒绝运行", dbName)
	}
	database.InitDB()
}
// newStubRuntime 构造不连库的纯内存实验运行时（SA/成本类纯函数单测用）。
func newStubRuntime() *ExperimentRuntime {
	rules := sandboxpkg.DefaultWorldRules()
	params := sandboxpkg.DefaultExperimentParams()
	return &ExperimentRuntime{
		Exp:      &model.Experiment{Id: 0, Seed: 1},
		Rules:    rules,
		Params:   params,
		rng:      rand.New(rand.NewSource(1)),
		Clock:    NewVirtualClock("2026-01-01 08:00:00", 10, 24),
		eventCfg: effectiveEventConfig(params, rules),
	}
}
