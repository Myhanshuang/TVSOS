# AGENT.md — TVSOS 项目上下文速览

> 供 AI 助手/新会话快速接手工作，避免重复交代背景。上次更新：2026-09（提交 58bc9c5 之后）。

## 项目是什么

TVSOS（运输车辆仿真调度系统）：模拟成都地区货运调度，SA 模拟退火批量派单 + 随机事件引擎（司机拒单/疲劳休息/极端天气/车祸维修），前端高德地图可视化 + 实时统计看板。

- 仓库：github.com/Myhanshuang/TVSOS，工作分支 `fullstack-refactor`，**只本地 commit，不 push**
- 后端 `tvsos-backend/`：Go (Gin + GORM + MySQL，Redis 可选、失败自动降级)，module 名 `github.com/kiritosuki/mover`
- 前端 `tvsos-frontend/`：Vue3 + Element Plus + ECharts + 高德地图 + Vite
- `ppt-tvsos-stats_ppt169_20260907/`：指标汇报 PPT 工程（不 git 跟踪）
- API 文档：根目录 `导入到apifox使用.json`（swagger 导出）；路由在 `internal/router/router.go`

## 常用命令

```
# 后端（Go 在 D:\sdk\go1.25.8\bin\go.exe）
cd tvsos-backend
go build -o mover-v.exe .      # 构建主程序
./mover-v.exe                  # 运行（读 config-dev.yml，端口 8088）

# 前端
cd tvsos-frontend
npm run dev                    # vite，端口 5173，仅 localhost
npm run build                  # 验证构建

# 数据重导入（若 DB 被测试清掉）
go run scripts/import_poi.go   # 1146 个成都真实 POI（poi_data/poi.json）
go run scripts/gen_mock_data.go # 20 辆川A车 + 10 货类 + 50 运单
```

前端 dev server 通过 vite proxy 把 `/api`→`http://127.0.0.1:8088`、`/ws`→后端 WS 代理。

## 红线（先读这里）

1. **绝不在真实数据库上运行 `go test ./internal/task/`** —— 测试连真实 MySQL 且会清空导入的车辆数据（历史上清掉过 20 辆车）。要测试请先备份或隔离库。
2. `go build ./...` 必失败：scripts/ 下的导入脚本有独立 main（双 main 属预期）。主程序用上面的单包构建命令。
3. `config-dev.yml` 被 gitignore（含数据库口令与随机事件参数），**新环境需手动提供**；`*.exe`、`dist/` 同样不入库。
4. 后端改动需 `go build -o mover-v.exe .` 重新构建并重启进程才生效。
5. 后端启动时 `EnsureDriversForVehicles` 自动为存量车辆补绑司机。

## 当前数据状态（约定口径）

- 1146 个成都真实 POI；20 辆车（川A·NORMAL1 … 川A·MINI004，置于真实 POI 坐标）；10 类货物；50 运单
- 随机事件配置在 `config-dev.yml [random_events]`：weather 0.0005/tick、accident 0.0002/tick、连续驾驶 4h 触发休息 20min、拒单 0.06（同单最多拒 2 次）
- 数据被清空的恢复路径：重跑两个导入脚本 → 删 driver/random_event 残留 → 重启后端

## 统计指标体系（全部已实时落地）

核心链路：`internal/task/stats.go CalculateGlobalStats()` → `realtime_stats.go BuildRealtimeDashboard()` → RealtimeSummary（WS `/ws/stats` 推送间隔 3s，前端轮询 15s 兜底）。

分视角面板 8 章（`statistics2.vue` 的 CHAPTERS）：M 匹配精准度 / U 利用效率 / T 吞吐响应 / A 算法层 / C 成本 / R 可靠性 / F 公平 / G 绿色。**无待接入占位**。

- M1-M4（P/R/F1/MRR）：参考答案为**条件最优**（固定其余指派、该单改派各车取整批成本最低者，`matching_quality.go`），逐单候选排名+ground truth 已回填 `matching_log`
- C1：SA 相对贪心首可行基线的成本改善%；A2 求解耗时（`solver_timing.go`）；A3 gap（`gap_evaluator.go`，≤4 单 ≤6 车才枚举，20 车队实际跳过）
- R1-R3（MTBF/MTTR/可用性）：观测窗口=最近一次 startup 事件；R4-R6 事故/维修/休息
- 全局优化框架：**4 主目标 C2↓ / U2↑ / U4↓ / T2 P95↓ + 2 软约束 F1 基尼≤0.4 / G2 碳排**；Cost A-F 定义在 `internal/task/cost.go`
- 已**移除**的指标（模拟环境无数据来源，勿加回）：S1-S3 周期生成、C3 ROI、A1 复杂度基准、F3 问卷满意度
- 指标文档（单一事实来源）：`tvsos-backend/docs/统计指标总结与全局优化对象.md`（docsify `_sidebar.md` 挂载）

## 调度与随机事件机制

- 派单：`order_task.go doCreateOrderTaskBatch` → `sa_scheduler.go assignBySimulatedAnnealing`（Cost F 唯一目标）→ `assignShipmentToVehicle`（写 OrderTask + MatchingLog + 预占载重）
- 随机事件 `random_event.go`：车祸→车辆 Maintenance + pause_until + system_event(fault)；天气→暂停移动；拒单→运单重回池子再派；疲劳→司机休息（车辆停驶）
- 车辆暂停机制已验证有效：pause_until 在未来时地图上车辆冻结、速度 0
- 运单生成：接口 `MockShipments`（随机起终点+货类）手动注入，**无周期生成器**

## 前端交互约定（用户明确要求，勿回退）

- `views/index.vue`：**横向** scroll-snap 切页，仅箭头按钮/键盘 ←→/顶部导航切换；**不用鼠标滚轮切页**（与地图缩放、页内纵向滚动冲突，已特意移除）
- 每个子页 section 内部纵向自由滚动
- 事件流与统计展示一律用**车牌号**（License 字段），不用「车辆#1」这类 ID 称呼
- `infoPanel.vue`：暂停中显示「等待中(休息/天气)」、速度 0.00、维修中状态
- `statistics2.vue` 新增 summary 字段需同步四处：`dynamicData` 声明、`initDynamicStats`（赋值风格）、`mapSnapshotToDynamicData`（对象字面量风格）、`applySnapshot`

## Windows/工具注意事项

- 仓库文本多为 CRLF；多行 edit 锚点内用 `\n` 即可正常匹配
- 前端 dev server 偶发 EBUSY（`.index.vue.*.tmpdir` 残留锁）：清掉该临时目录再重启
- 涉及 ppt-master 等 Python 技能时用 `python` 而非 `python3`
- 长时间运行的服务用后台任务管理，注意端口占用（8088 后端 / 5173 前端 / 5050 PPT 预览）

## Git 约定

- 中文提交信息，conventional 前缀：`feat:` / `fix:` / `docs:`
- 只 commit 不 push；近期提交：af9a22f（指标落地清理）、58bc9c5（P2 匹配质量）

## PPT 产物（可选参考）

- `ppt-tvsos-stats_ppt169_20260907/exports/*.pptx`：10 页指标汇报（基于指标文档生成）
- 该目录不 git 跟踪；指标文档更新后 PPT 内容可能滞后，需重新生成时走 ppt-master 流程

## 服务状态（默认停止）

服务当前未运行。启动：先起后端 `mover-v.exe`（等 DB/司机绑定日志就绪），再起前端 `npm run dev`，访问 http://localhost:5173。注意：重启后端会重置 M 系/A2/R1-R3 的进程内窗口累计（DB 表数据保留）。
