"use strict";
const H = require("./docx_helpers.js");
const AlignmentType = require("docx").AlignmentType;
var out = [];

// ================= chapter 2 =================
var ch2 = [];
ch2.push(H.H1("第二章  存在问题与解决方案"));
ch2.push(H.H2("2.1 存在的主要问题"));
ch2.push(H.P([H.b("（1）货物材料需求发生链条尚未实现。"), H.t("任务书 1.1 要求实现货物材料加工场景的吨位/方量变化趋势与至少一条需求发生链条（如“原木—木材—板材—家具”）。当前订单流为各货类独立的泊松到达，货类之间没有上下游传导关系，“家具厂的订单会拉动板材厂”这类供应链因果尚未建模。这是当前与任务书要求差距最明确的一项。")]));
ch2.push(H.P([H.b("（2）事件与状态数量距“优”档有缺口。"), H.t("现有司机行为状态 3 种（可用、拒单、休息中）加维修联动，随机事件 4 类（拒单、疲劳休息、极端天气、车祸）。任务书评价标准“优”档要求司机行为状态 5 种以上、随机事件 5 种以上，还差 1～2 类（缺货、货损、交通拥堵等）。")]));
ch2.push(H.P([H.b("（3）多目标机制形态待丰富。"), H.t("现有加权标量化与主/软约束分层两种机制已满足“至少 2 种”的下限，但缺乏帕累托前沿类机制，无法回答“权重 λ 变化下解集如何权衡”这一更深层问题；碳排目前只在结果层计量，未形成对派单的约束力。")]));
ch2.push(H.P([H.b("（4）评价口径的固有局限。"), H.t("全局精确最优在 20 辆车队下组合爆炸，M1-M4 采用条件最优口径（弱于全局最优）；A3 精确 gap 仅在小规模时启用。评价体系存在“以弱口径自证”的理论局限，需在报告与答辩中如实说明。")]));
ch2.push(H.H2("2.2 解决方案"));
ch2.push(H.P([H.b("（1）需求链条（对应问题 1）："), H.t("设计如图 2-1 的传导规则，在数据生成器实现：基础需求按泊松独立到达；上游节点订单成交后以概率 β 经加工延迟 L 触发下游补货订单；下游吨位按产出率 α 折算（体现加工损耗）；链条拓扑与 α/β/L 参数写入世界规则 TOML，生成器版本升 v3 并保持逐字节可复现。POI 库已含家具/建材/五金类真实工厂与市场（1146 个成都 POI），链条起点/终点可直接落在真实坐标上。")]));
ch2.push(H.figure("fig6_chain.png", 500));
ch2.push(H.caption("图 2-1  货物材料需求发生链条设计（后续实现方案）"));
ch2.push(H.P([H.b("（2）事件补齐（对应问题 2）："), H.t("新增三类事件——缺货（装货节点库存不足，延迟装货形成额外等待）、货损（运输中部分货损，送达吨位折减，影响吞吐与成本）、交通拥堵（在分时段速度系数基础上叠加随机拥堵路段，降速不改道）。加上现有 4 类达 7 类；司机行为状态增加“请假”后达 5 种，满足“优”档数量要求。")]));
ch2.push(H.P([H.b("（3）帕累托机制（对应问题 3）："), H.t("在加权法之外实现 ε-约束法变体：固定 C2 为主目标，将 U4 空驶率、T2 P95 作为 ε 约束边界，扫描 ε 网格生成近似帕累托前沿；实验参数开放 λ 权重与 ε 边界，用同种子批次做受控对比。碳排约束：G2_carbon_max 收紧时向 Cost 追加超限惩罚项。")]));
ch2.push(H.P([H.b("（4）口径补强（对应问题 4）："), H.t("小批次保留精确枚举 gap 作“锚点”校验条件最优口径的偏差；报告与答辩中所有结论均注明口径与局限。")]));

// ================= chapter 3 =================
var ch3 = [];
ch3.push(H.H1("第三章  前期任务完成度与后续实施计划"));
ch3.push(H.H2("3.1 任务书要求完成度对照"));
ch3.push(H.P("按任务书主要任务逐项对照当前实现状态如表 3-1 所示。"));
ch3.push(H.tableCaption("表 3-1  任务书主要任务完成度对照"));
ch3.push(H.threeLineTable([
  [
    { lines: "任务书条目", w: 14 },
    { lines: "要求", w: 22 },
    { lines: "本阶段实现", w: 50 },
    { lines: "状态", w: 14 },
  ],
  [
    { lines: "1.1 需求链条" },
    { lines: "吨位/方量变化趋势，至少 1 条链条" },
    { lines: "订单流泊松到达已实现；链条传导规则已设计（图 2-1），编码列入计划" },
    { lines: "部分完成", align: AlignmentType.CENTER },
  ],
  [
    { lines: "1.2 司机行为状态" },
    { lines: "至少 2 种，数学概率基准" },
    { lines: "拒单（p=0.06×等级修正）、疲劳休息（4h/20min）、维修联动；增加请假状态后达 5 种" },
    { lines: "基本完成", align: AlignmentType.CENTER },
  ],
  [
    { lines: "1.3 运输随机事件" },
    { lines: "至少 2 种" },
    { lines: "极端天气、车祸 2 类已实现（另有司机侧拒单/休息）；补齐缺货/货损/拥堵后达 7 类" },
    { lines: "基本完成", align: AlignmentType.CENTER },
  ],
  [
    { lines: "1.4 实时统计" },
    { lines: "对优化目标实时统计" },
    { lines: "八维指标体系 + WS 3s 推送 + 每 6 tick 快照落库 + 匹配质量/可靠性评价" },
    { lines: "已完成", align: AlignmentType.CENTER },
  ],
  [
    { lines: "2 全局多目标优化" },
    { lines: "多目标机制至少 2 种" },
    { lines: "SA 全局批量调度；机制一：Cost A–F 加权标量化；机制二：4 主目标 + 2 软约束分层与同种子受控对比；帕累托机制列入计划" },
    { lines: "基本完成", align: AlignmentType.CENTER },
  ],
]));
ch3.push(H.spacer(120));
ch3.push(H.P("对照任务书“优”档标准（司机状态 ≥5 种、随机事件 ≥5 种、多目标机制 >3 种），当前处于“良”向“优”推进的中段：实时统计与全局优化主链已达“优”档工作量，缺口集中在需求链条与事件/机制数量，方案与排期均已明确。"));
ch3.push(H.H2("3.2 后续实施计划"));
ch3.push(H.tableCaption("表 3-2  后续实施计划"));
ch3.push(H.threeLineTable([
  [
    { lines: "时间", w: 12 },
    { lines: "工作项", w: 22 },
    { lines: "关键产出", w: 66 },
  ],
  [
    { lines: "第 1 周", align: AlignmentType.CENTER },
    { lines: "需求发生链条" },
    { lines: "生成器 v3（α/β/L 传导）、TOML 规则、链条订单回归测试" },
  ],
  [
    { lines: "第 2 周", align: AlignmentType.CENTER },
    { lines: "事件补齐" },
    { lines: "缺货/货损/拥堵三类事件 + 请假状态，事件数达 7、状态数达 5" },
  ],
  [
    { lines: "第 3 周", align: AlignmentType.CENTER },
    { lines: "帕累托多目标" },
    { lines: "ε-约束扫描、近似帕累托前沿、λ/ε 参数面板、碳排超限惩罚接入" },
  ],
  [
    { lines: "第 4 周", align: AlignmentType.CENTER },
    { lines: "实验与结题" },
    { lines: "多组参数批次实验、对比分析、结题报告与答辩材料" },
  ],
]));
ch3.push(H.spacer(120));
ch3.push(H.H2("3.3 团队分工"));
ch3.push(H.P("本阶段的方案设计（指标体系选型、优化对象框架、沙箱总体架构、展示规范）由全体小组集体讨论完成，编码实现按六个工作包推进："));
ch3.push(H.P([H.b("丁宁：统计计算引擎与指标落地（后端）："), H.t("统计链路、M/U/T/A/C/R/F/G 八维指标、匹配质量评价；")]));
ch3.push(H.P([H.b("李韦成：司机与随机事件引擎（后端）："), H.t("司机对象、四类事件状态机、统一暂停机制；")]));
ch3.push(H.P([H.b("胡天琦：运行期指标展示（前端）："), H.t("地图数据覆盖层、八组卡片、事件流时间线；")]));
ch3.push(H.P([H.b("万飞翔：沙箱数据模型与鉴权（后端）："), H.t("四级模型、维度隔离迁移、双层 TOML、登录鉴权；")]));
ch3.push(H.P([H.b("胡杨：沙箱引擎与全套 API（后端）："), H.t("实验运行时、虚拟时钟、确定性生成器、对比接口；")]));
ch3.push(H.P([H.b("华观中：沙箱页面流与结果工具链（前端）："), H.t("沙箱/实验页面流、参数分栏 Diff、报告导出。")]));

// ================= references =================
var refs = [];
refs.push(H.H1("参考文献"));
refs.push(H.refP(1, "进阶式挑战性综合项目 III 课题任务书——运输车辆调度优化与仿真. 电子科技大学信息与软件工程学院, 2026."));
refs.push(H.refP(2, "GIN TEAM. Gin Web Framework[EB/OL]. (2015-04-04)[2026-09-25]. https://github.com/gin-gonic/gin."));
refs.push(H.refP(3, "APACHE SOFTWARE FOUNDATION. Apache ECharts[EB/OL]. (2013)[2026-09-25]. https://echarts.apache.org/."));
refs.push(H.refP(4, "IETF. RFC 6455: The WebSocket Protocol[S/OL]. (2011-12)[2026-09-25]. https://datatracker.ietf.org/doc/html/rfc6455."));
refs.push(H.refP(5, "KIRKPATRICK S, GELATT C D, VECCHI M P. Optimization by Simulated Annealing[J]. Science, 1983, 220(4598): 671-680."));
refs.push(H.refP(6, "JINZHU Z. GORM: The fantastic ORM library for Golang[EB/OL]. [2026-09-25]. https://gorm.io."));
refs.push(H.refP(7, "YOU E. Vue.js: The Progressive JavaScript Framework[EB/OL]. (2014)[2026-09-25]. https://vuejs.org."));
refs.push(H.refP(8, "高德开放平台. Web 服务 API 文档[EB/OL]. [2026-09-25]. https://lbs.amap.com."));

out = ch2.concat(ch3, refs);
module.exports = out;
