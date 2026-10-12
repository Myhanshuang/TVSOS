"use strict";
const H = require("./docx_helpers.js");
var ch1 = [];
ch1.push(H.H1("第一章  综合设计的进展情况"));

ch1.push(H.H2("1.1 针对工程问题的方案设计"));
ch1.push(H.P("本课题承接综合设计 II 阶段的成果——基于 Go + Gin + GORM 的调度后端、五维代价模型（A 直接成本、B 时效成本、C 效率与公平成本、D 平台资源成本、E 风险成本，经 F 综合加权合成）、模拟退火全局调度算法、高德地图路径规划驱动的物理仿真引擎，以及 Vue3 + ECharts 的可视化大屏。针对任务书提出的“优化仿真参数使其更贴近真实场景、实现多目标全局优化算法、实时统计优化目标”三项主要任务，本阶段我们完成了系统的总体架构重构，将原有“单一仿真进程”升级为“实验沙箱 + 受控实验”的四层架构（图 1-1）。"));
ch1.push(H.P([H.t("四层职责划分如下。"), H.b("展示层"), H.t("：Vue3 + Element Plus + ECharts + 高德地图，包含沙箱管理、实验配置（表单与 TOML 双向编辑）、地图大屏（可拉入数据覆盖层）、任务管理、实验结果与实验对比页面；"), H.b("服务层"), H.t("：Go + Gin，REST API 与 WebSocket 双通道，登录鉴权中间件保护全部业务路由；"), H.b("仿真运行层"), H.t("：ExperimentRuntime 以单实验串行、单线程确定性 tick 驱动，内含虚拟时钟、SA 调度、随机事件引擎、全局统计引擎与种子随机源；"), H.b("数据与生成层"), H.t("：确定性数据生成器在仿真之外预生成订单流与事件脚本，MySQL 中沙箱元数据表与 9 张运行表按 experiment_id 实现实验实例级数据隔离。")]));
ch1.push(H.P([H.t("架构重构遵循三条设计原则。"), H.b("可复现"), H.t("：一切随机性收敛到单一种子 RNG，同种子同参数必得到逐位一致的结果；"), H.b("可观测"), H.t("：八维指标体系（M/U/T/A/C/R/F/G）对调度质量、运力效率、时效、算法性能、成本、可靠性、公平性、碳排做实时统计；"), H.b("可扩展"), H.t("：事件概率、司机行为参数、经济与交通环境全部进入配置或双层 TOML，新增事件类型不需改动调度与统计代码。")]));
ch1.push(H.figure("fig1_arch.png", 540));
ch1.push(H.caption("图 1-1  系统总体架构（四层）"));

ch1.push(H.H2("1.2 针对工程问题的推理分析与具体实现"));
ch1.push(H.P("在明确总体架构后，本节按“司机行为真实化 → 运输过程随机事件 → 全局多目标优化 → 实时统计评价 → 实验可复现”的顺序，说明关键模块的推理过程与代码实现。"));

ch1.push(H.H3("1.2.1 司机对象与概率化行为状态机"));
ch1.push(H.P("真实货运行业有两个经验事实：其一，司机对距离远、价格不划算的订单存在拒单行为，且驾驶经验越丰富拒单率与事故率越低；其二，法规要求连续驾驶 4 小时须休息不少于 20 分钟。据此，系统建立一车一司机的司机对象（driver 表：姓名、电话、驾驶证号、驾驶等级、疲劳累计、拒单计数、绑定车辆），驾驶等级（2～5 级）进入概率修正。"));
ch1.push(H.P("行为均以数学概率为基准：拒单按 p = 0.06 × 等级修正因子（等级从低到高对应 1.4→0.7）逐次判定，同一运单最多被拒 2 次，超过后强制接单，防止“订单永远无人接”的死循环；疲劳按连续驾驶小时数累计，达 4h 触发 20min 强制休息。拒单不是简单丢弃订单，而是一套完整回退链路：删除 Planned 任务 → 释放车辆预占载重 → 运单回退 Sleeping → 下一轮调度重新进入分配池 → 车辆状态由 Ready 回退为 Free。这一设计复用了既有的重派机制，使拒单事件对调度系统表现为“可消化的扰动”而非异常。"));
ch1.push(H.codeCaption("代码 1-1  司机拒单概率判定（random_event.go 节选）"));
ch1.push.apply(ch1, H.codeBlock([
  "// 发车前判定：司机有概率拒绝接单（等级修正 + 同单上限防死循环）",
  "refuseCount, _ := repository.CountRefuseByShipment(shipment.Id)",
  "if refuseCount >= int64(cfg.RefuseMaxPerShipment) {",
  "    return false // 同一运单被拒超过上限后强制接单",
  "}",
  "if rt.Draw() >= cfg.DriverRefuseProb*levelFactorOf(cfg, driver.Level) {",
  "    return false // 未触发拒单",
  "}",
  "// 触发拒单：删除 Planned 任务、释放车辆预占载重、运单回退 Sleeping 等待重派",
]));
ch1.push(H.P("指标联动方面，拒单拉长订单响应长尾（T2 P95）、降低一次接受率（M 系），休息降低在线运力（U3）——司机行为由此成为被观测、被统计的一等公民，而非脚本化的背景噪声。"));

ch1.push(H.H3("1.2.2 运输过程随机事件引擎与统一暂停机制"));
ch1.push(H.P("在司机行为之外，运输过程引入两类随机事件：极端天气（p=0.0005/tick，暂停 0.5～2h，模拟暴雨封路与限行）与车祸（p=0.0002/tick × 等级因子，车辆进入新增的“维修中”状态 1～6h，在途任务挂起）。至此系统共有四类随机事件（拒单、疲劳休息、极端天气、车祸），概率与时长区间全部可配置。"));
ch1.push(H.P([H.t("设计上的关键取舍是"), H.b("统一暂停机制"), H.t("：三类“时间冻结”（疲劳休息/天气/维修）共用一个检查点，moveVehicle 每 tick 开头以虚拟时间比较 pause_until，未到期则本 tick 位置冻结、速度为 0，到期自动恢复。这把“事件的影响”统一抽象为“可用运力的时间窗收缩”，对调度器与统计引擎完全透明，也使暂停行为可在前端地图上被直观验证（车辆冻结、速度 0.00）。")]));
ch1.push(H.codeCaption("代码 1-2  统一暂停判定（vehicle.go 节选）"));
ch1.push.apply(ch1, H.codeBlock([
  "// moveVehicle 开头：随机事件暂停窗口内（休息/极端天气）不推进，虚拟时间比较",
  "now := rt.Clock.Now()",
  "if isVehiclePaused(latestVehicle, now) {",
  "    return // pause_until 未到期：车辆位置冻结、速度为 0",
  "}",
  "// 天气/车祸事件由预生成脚本驱动（consumeDueEvents），此处不再现场摇号",
]));
ch1.push(H.P("每次事件写入 random_event 表（类型、车辆、司机、运单、严重度、时长、描述），前端事件流按车牌实时展示（类型着色、未解决空心），并进入 R 系可靠性指标：MTBF（平均无故障时间）、MTTR（平均修复时长）与可用性 = 1 − 停机时长 / (车队规模 × 观测窗口)，观测窗口为实验生命周期（虚拟时钟起点至今）。图 1-2 汇总了司机行为状态、运输随机事件状态机与统一暂停机制的全部参数。"));
ch1.push(H.figure("fig2_events.png", 540));
ch1.push(H.caption("图 1-2  司机行为状态与运输随机事件状态机（含概率参数）"));

ch1.push(H.H3("1.2.3 全局多目标优化机制"));
ch1.push(H.P([H.t("第一种多目标机制是承接 II 阶段的"), H.b("加权标量化"), H.t("：将调度决策抽象为 Cost = λ1·A + λ2·B + λ3·C + λ4·D + λ5·E 的单一可排序分数，代价项与任务书五类优化目标一一对应：车辆总体等待时间（A 的等待项与 B 的货物加权等待）、空载里程（A 的空驶项与 C 的空驶率）、车辆利用率（C 的等待运输比与 D 的运力缺口惩罚）、总运输达成吨位（D 的运力结构惩罚与吞吐统计）、碳排放（G 系）。SA 算法以 Metropolis 准则接受劣解跳出局部最优，通过 Swap/Move 邻域扰动迭代搜索全局低成本指派。")]));
ch1.push(H.codeCaption("代码 1-3  模拟退火 Metropolis 接受准则（sa_scheduler.go 节选）"));
ch1.push.apply(ch1, H.codeBlock([
  "// 邻域扰动 + Metropolis 接受准则：以概率接受劣解，跳出局部最优",
  "_, nextAssigned := generateSANeighbor(rt, currentAssigned, len(vehicles))",
  "nextCost := evaluateStateCost(nextAssigned, shipments, vehicles, stats, ...)",
  "if nextCost < currentCost || math.Exp((currentCost-nextCost)/temp) > rt.Draw() {",
  "    currentAssigned = nextAssigned",
  "    currentCost = nextCost",
  "    if currentCost < bestCost {",
  "        copy(bestAssigned, currentAssigned)",
  "        bestCost = currentCost",
  "    }",
  "}",
]));
ch1.push(H.P("真实化方面，沙箱世界规则向系统注入经济环境：A 项追加货币化里程成本（油价 × 百公里油耗 + 每公里过路费 + 人力成本），使成本从抽象分数落到可解释的货币量纲；交通环境提供分时段速度系数（高峰降速），影响每 tick 推进的里程与疲劳累计，也为后续拥堵事件提供了机制雏形。"));
ch1.push(H.P([H.t("碳排放方面，G1/G2 按车型排放因子（空驶/载重分列）乘累计里程计算总碳排与吨公里碳排强度。当前碳排以软约束形式进入实验参数（G2_carbon_max 上限），主要在结果层计量与呈现，"), H.b("约束式惩罚接入代价函数列入后续计划"), H.t("（见 2.2 节）。")]));
ch1.push(H.codeCaption("代码 1-4  碳排放计算（stats.go 节选）"));
ch1.push.apply(ch1, H.codeBlock([
  "// G1/G2：按车型排放因子 × 空/重载里程累计总碳排，吨公里口径核算强度",
  "for _, v := range vehicles {",
  "    fEmpty, fLoad := defaultEmissionFactors(v.Tybe)",
  "    totalCarbon += v.EmptyMileage*fEmpty + v.Distance*fLoad",
  "}",
  "s.CarbonStats.CarbonPerTonKm = totalCarbon / s.TransportWorkStats.TotalTonDistance",
]));
ch1.push(H.P([H.t("第二种多目标机制是"), H.b("主目标/软约束分层框架"), H.t("：不追求单一分数，而是将优化对象显式分层为 4 项主目标（C2 单位成本↓、U2 载重利用率↑、U4 空驶率↓、T2 响应 P95↓）与 2 项软约束（F1 负载基尼≤0.4、G2 吨公里碳排），实验结果页按此六项做主指标对比；配合同种子跨参数批次的对比实验（参数 TOML 分栏 Diff + 结果对比页），等价于对多目标权衡的受控敏感性实验（图 1-3）。帕累托类机制（ε-约束/NSGA-II）尚未实现，作为“优”档的增强项列入计划。")]));
ch1.push(H.figure("fig4_metrics.png", 540));
ch1.push(H.caption("图 1-3  多目标优化对象与八维指标体系支撑关系"));

ch1.push(H.H3("1.2.4 优化目标的实时统计与匹配质量评价"));
ch1.push(H.P("实时统计链路为：CalculateGlobalStats（每虚拟 tick 全量计算）→ BuildRealtimeDashboard（汇总为 RealtimeSummary）→ WebSocket /ws/stats 每 3 真实秒推送（前端 15s 轮询兜底）；每 6 虚拟 tick（默认 1 虚拟分钟）将十七项核心指标写入 metric_snapshot 表（按 experiment_id 隔离），支撑实验结束后的收敛曲线与分位数分析。任务书“实现对优化目标的实时统计”以此完成。前端层面，运行期指标以地图数据覆盖层呈现（运行健康、运力效率趋势、Top3 最值榜、车辆状态构成、调配效率五维雷达、最终优化指标值与随机事件流），实验结束后结果页按 M/U/T/A/C/R/G + 事件积压八组卡片呈现终态。"));
ch1.push(H.P([H.t("全局优化算法的质量不能只看成本曲线自说自话。我们为调度质量建立了基于"), H.b("条件最优参考答案"), H.t("的评价体系：固定一批 SA 派单的其余指派不变，将某一运单改派给任一候选车后重新评估整批成本，取最低者为该单的条件最优车（ground truth）。相比全局精确枚举（20 辆车队组合爆炸），条件枚举复杂度仅 O(运单数 × 车辆数)，任意批次规模均可实时完成。据此统计 M1 精确率（选中恰为条件最优）、M2 召回率（分母含未派单运单）、M3 F1、M4 MRR（选中车排名倒数均值），并把每单 Top8 候选排名回填匹配日志供离线复盘（图 1-4）。")]));
ch1.push(H.codeCaption("代码 1-5  条件最优参考答案评估（matching_quality.go 节选）"));
ch1.push.apply(ch1, H.codeBlock([
  "// 固定其余指派，逐单改派各候选车评估整批成本，取最低者为 ground truth",
  "copy(variant, saIdx)",
  "for vi := range vehicles {",
  "    variant[i] = vi",
  "    cands = append(cands, qualityCand{vi,",
  "        evaluateStateCost(variant, shipments, vehicles, stats, ...)})",
  "}",
  "sort.Slice(cands, func(a, b int) bool { return cands[a].cost < cands[b].cost })",
  "// 排名首位即条件最优车；SA 实际选中车的位次进入 M1/M4 统计",
]));
ch1.push(H.P("在算法性能维度，同批次重放“贪心首可行”基线，C1 = (基线成本 − SA 成本)/基线成本 × 100%，直接量化 SA 相对朴素策略的改善幅度；A2 记录每批求解耗时（批数/均值/峰值）；A3 在小规模（≤4 单 ≤6 车）时做精确枚举 gap 估算。可靠性维度（R1-R3 及事件计数）已在 1.2.2 节说明。至此，调度质量的评价形成“在线指标实时可见 + 离线日志可复盘”的双层证据链。"));
ch1.push(H.figure("fig3_quality.png", 540));
ch1.push(H.caption("图 1-4  基于条件最优参考答案的匹配质量评价闭环"));

ch1.push(H.H3("1.2.5 实验沙箱与确定性复现"));
ch1.push(H.P([H.t("仿真实验若不可复现，则一切“算法 A 优于算法 B”的结论都不成立。本阶段我们把“实验”本身作为设计对象，建立四级模型："), H.b("User（研究者）→ Sandbox（世界设定：POI/车队/货类白名单）→ ExperimentScheme（参数模板）→ Experiment（批次：参数快照 + 种子 + 结果）"), H.t("，参数采用双层 TOML（沙箱世界规则 / 实验参数）描述（图 1-5）。")]));
ch1.push(H.P("三件事保证严格复现。其一，虚拟时钟：仿真时间 = 起点 + tick 数 × 步长，与真实时钟解耦，倍速只改变真实世界的 tick 触发频率；其二，单线程确定性 tick：主循环每 tick 按固定顺序执行“释放订单→SA 派单→消费事件脚本→恢复维修/休息→移动→快照”，全部随机性来自单一种子 RNG，SA 的随机邻域、司机等级、事件触发同源；其三，确定性数据生成器：在仿真系统之外按“世界规则 + 实验参数 + 种子”预生成订单流（泊松/均匀到达）与随机事件脚本，同种子同参数输出逐字节一致（单测断言），并以 DataGeneratorVersion（v2）防止旧种子误导入。"));
ch1.push(H.codeCaption("代码 1-6  虚拟时钟（virtual_clock.go 节选）"));
ch1.push.apply(ch1, H.codeBlock([
  "// 虚拟时间 = 起点 + tick 数 × 步长；倍速只改变真实世界的 tick 触发频率",
  "func (c *VirtualClock) Now() time.Time {",
  "    return c.start.Add(time.Duration(c.tick.Load()*c.tickSeconds) * time.Second)",
  "}",
]));
ch1.push(H.P("复现性已得到实证：同种子两次完整实验的调度总成本逐位一致（35025.80213604554）。工程配套上，9 张运行表增加 experiment_id 维度列（0=沙箱模板态、>0=实验实例）实现数据隔离；实验启动时从车队模板实例化专属车辆与司机；登录鉴权与内存 token 保护全部业务路由与 WebSocket。测试环境方面，task 包测试强制读独立的 config-test.yml 测试库，解决了误伤生产数据库的后顾之忧；生成器与纯函数模块均有单元测试覆盖。"));
ch1.push(H.figure("fig5_sandbox.png", 540));
ch1.push(H.caption("图 1-5  实验沙箱流程与复现机制"));

ch1.push(H.H2("1.3 知识技能学习情况"));
ch1.push(H.P([H.t("在本阶段开发中，团队成员在工程实践上获得了显著提升："), H.b("1. 确定性系统设计"), H.t("：理解了“可复现是仿真实验的生命线”，掌握 RNG 消费序控制、虚拟时钟与单线程 tick 的组合技术；"), H.b("2. 测试驱动开发与测试环境构造"), H.t("：独立测试库隔离、纯函数桩运行时、字节级一致性断言；"), H.b("3. 概率建模"), H.t("：泊松订单流、逐 tick 伯努利事件、等级修正因子的工程化落地；"), H.b("4. 多目标优化理论向工程的转化"), H.t("：标量化、主/软约束分层，并将检索评价的 P/R/F1/MRR 思想迁移到调度质量评价；"), H.b("5. 前端工程"), H.t("：WebSocket 实时推送、覆盖层交互动画、自实现 LCS 行级分栏 Diff；"), H.b("6. 数据建模"), H.t("：维度列隔离、参数快照、种子导入导出的可重放实验设计。")]));

module.exports = ch1;
