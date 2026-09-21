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

1. ~~绝不在真实数据库上运行 go test~~ **已解除**（2026-09-08）：task 包测试强制读 config-test.yml（独立测试库，库名须含 test），缺失即 Skip，见 test_helper_test.go。
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

实时展示已改为地图覆盖层（`components/mapDataOverlay.vue`，非弹窗三整板）：左整板全高（运行健康图：吞吐/积压/P95+事件标记；运力效率趋势：U2/U4/U3+状态堆叠带）向右滑入；右整板全高（Top3 最值榜：载货最重/载货时长，点击弹完整列表；车辆状态饼图；调配效率五维雷达：里程利用率/装载率/派单率/货物流动率/运力利用率，后端 radar 字段 0~100）向左滑入；底整板（最终优化指标值 F/C2/C1/A3 + 随机事件流）向上滑入。三整板完全打开后边缘完全贴合零空隙。完整指标体系（M/U/T/A/C/R/F/G 八章）保留在实验结果页。

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
- 数据覆盖层：`dataPanel` Store 控制显隐（导航“显示数据”按钮，header.vue handleToggleData），与 infoPanel 双向互斥（数据面板开→点车辆/POI 时覆盖层先渐出、infoPanel 延迟 0.2s 进入；infoPanel 开→点“显示数据”先收回 infoPanel、0.3s 后再拉入数据整板，300ms 内连点视为取消；覆盖层不动地图尺寸）

## Windows/工具注意事项

- 仓库文本多为 CRLF；多行 edit 锚点内用 `\n` 即可正常匹配
- 前端 dev server 偶发 EBUSY（`.index.vue.*.tmpdir` 残留锁）：清掉该临时目录再重启
- 涉及 ppt-master 等 Python 技能时用 `python` 而非 `python3`
- 长时间运行的服务用后台任务管理，注意端口占用（8088 后端 / 5173 前端 / 5050 PPT 预览）

## Git 约定

- 中文提交信息，conventional 前缀：`feat:` / `fix:` / `docs:`
- 只 commit 不 push；近期提交：af9a22f（指标落地清理）、58bc9c5（P2 匹配质量）



---

## 沙箱系统（2026-09-08 落地，当前主流程）

概念：`User(研究者) → Sandbox(世界设定) → ExperimentScheme(参数模板) → Experiment(批次:参数快照+种子+结果)`。
默认主页是沙箱管理页（/admin/sandboxes），登录（admin/admin123）→ 沙箱 → 新建实验（参数面板/TOML/种子导入）→ 生成数据 → 开始实验 → 地图+任务管理两屏（地图页可“显示数据”拉入凹字形覆盖层） → 到达固定虚拟时长自动完成 → 结果页；对比页 /admin/sandbox/:id/compare。

### 后端要点
- **单实验串行**：`internal/task/experiment_runtime.go` ExperimentRuntime，同一时刻仅一个活跃实验；POST /experiments/:id/start|stop 控制生命周期；服务重启=当前实验 abort（AbortStaleExperiments）。
- **虚拟时钟（严格复现）**：`virtual_clock.go`，虚拟时间=起点+tick×步长；pause_until/阶段时间戳/事件时刻/快照时刻全是虚拟时间；倍速只改真实 tick 频率。
- **单线程 tick**：masterLoop 每 tick 顺序执行 释放订单→SA派单→消费事件→恢复→移动→快照，RNG 消费序确定 ⇒ 同种子逐 tick 一致（已验证：总成本 35025.80213604554 两次一致）。
- **种子 RNG**：rt.rng=rand.New(rand.NewSource(seed))；SA/司机等级/拒单全走它；天气/车祸由**预生成脚本**（experiment_event 表）到点消费。
- **数据生成器** `internal/sandbox/gen`：Generate 纯函数（摇号序固定：司机等级→订单流泊松/均匀→天气/车祸逐 tick 摇号，车祸带 level_factor 等级修正；等级表存 experiment.driver_levels 供实例化同源使用），同种子逐字节一致（generator_test.go 单测）；`internal/sandbox/params.go` 是世界规则/实验参数 TOML 模式与默认值，DataGeneratorVersion（当前 v2）供种子导入校验。
- **数据隔离**：9 张运行表带 experiment_id（0=沙箱模板/全局，>0=实验实例）；实验启动时从 sandbox.fleet_template 实例化本实验专属 vehicle/driver 行；scoped_repo.go 全是按实验过滤的查询变体。
- **结果层**：experiment_result（summary_json 全量 + primary_metrics 主目标 C2/U2/U4/T2/F1/G2 + convergence SA 收敛曲线）；订单/事件脚本在 experiment_order/experiment_event。
- **登录**：简单演示级——user 表+bcrypt+内存 token（24h，重启失效），/auth/login|register|profile|logout；middleware.AuthRequired 挂全部业务路由（WS 走 ?token=）；除 /auth/login|register 与 /swagger 外全保护。
- **经济/交通环境**：cost.go SetEconomyRules（实验启动注入，A 项追加货币化里程成本=油价+过路费+人力）；rt.SpeedKmhAt 分时段速度系数（rush_hours × rush_speed_factor，同时决定每 tick 推进点数与时长/疲劳累计）。
- R1-R3 可靠性/T1 吞吐观测窗口改为**实验生命周期**（虚拟时钟起点至今），不再用最近一次 startup 事件。
- 指标快照改由运行时每 6 虚拟 tick 写入（原 MetricSnapshotLoop 已删；BroadcastStatsLoop 改为运行时的 3s 真实秒展示推送，不影响仿真状态）。
- MockShipments 接口与前端 mock 按钮已删除（由数据生成器取代）。

### 前端要点
- 新页面：views/sandboxList.vue、sandboxDetail.vue（方案+实验记录+参数导入导出）、experimentResult.vue、experimentCompare.vue、components/experimentConfig.vue（表单↔TOML 双向，实验与方案新建共用——方案走 scheme-mode 属性：多描述字段、仅创建不生成）。导出=参数 TOML 独立文件浏览器下载（Blob+URL.createObjectURL，不再走剪贴板 JSON 种子包）；导入双格式：JSON 种子包走 /import，纯 TOML 文本走 createExperiment 按参数建实验。
- index.vue 两屏（地图/任务管理）是实验视图 /admin/experiment/:id（原独立统计分析屏已删除，并入地图覆盖层 mapDataOverlay.vue）；header.vue 按场景切换导航（实验侧有地图/任务管理/显示数据开关/倍速/虚拟时钟进度条/结束实验；沙箱侧只有沙箱管理入口），退出登录真实化。
- WS 连接统一经 withToken() 附加 ?token=（mapDataOverlay.vue / vehiclePolling.js）。
- request.js 无条件携带 Authorization: Bearer，401 自动清 token 回登录页。

### 测试红线更新
- task 包测试现在强制走 **config-test.yml**（库名必须含 test 字样，否则 Skip/拒绝），解除"跑 go test 清生产库"红线。纯内存桩运行时 newStubRuntime() 供纯函数单测。
- 生成器单测 `go test ./internal/sandbox/...` 与 task 包纯单测可安全跑。

## PPT 产物（可选参考）

- `ppt-tvsos-stats_ppt169_20260907/exports/*.pptx`：10 页指标汇报（基于指标文档生成）
- 该目录不 git 跟踪；指标文档更新后 PPT 内容可能滞后，需重新生成时走 ppt-master 流程

## 服务状态（默认停止）
注意：沙箱化后**仿真不再随进程常驻**，服务启动后须在沙箱页创建实验并"生成数据→开始实验"，地图才有数据。

服务当前未运行。启动：先起后端 `mover-v.exe`（等 DB/司机绑定日志就绪），再起前端 `npm run dev`，访问 http://localhost:5173。注意：重启后端会重置 M 系/A2/R1-R3 的进程内窗口累计（DB 表数据保留）。
