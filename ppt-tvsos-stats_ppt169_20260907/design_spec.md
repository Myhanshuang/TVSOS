# tvsos-stats - Design Spec

> Human-readable design narrative. Machine-readable execution contract: `spec_lock.md` — Executor re-reads it before every SVG page.

## I. Project Information

| Item | Value |
| ---- | ----- |
| **Project Name** | tvsos-stats |
| **Canvas Format** | PPT 16:9 (1280x720) |
| **Page Count** | 10 |
| **Design Style** | data-journalism（数据报告风格，蓝青科技配色） |
| **Target Audience** | 项目评审 / 技术决策者 |
| **Use Case** | 内部技术汇报——指标体系设计与优化方向陈述 |
| **Delivery Purpose** | presentation（一页一主题，大字少文，汇报用） |
| **Content Strategy** | balanced——重构为金字塔叙事并保留全部实质内容，不增删事实 |
| **Created Date** | 2026-09-07 |

---

## II. Canvas Specification

| Property | Value |
| -------- | ----- |
| **Format** | PPT 16:9 |
| **Dimensions** | 1280x720 |
| **viewBox** | `0 0 1280 720` |
| **Margins** | 左右 64px，上下 56px |
| **Content Area** | 1152x608 |

---

## III. Visual Theme

### Theme Style

- **Mode**: pyramid（结论先行——每页标题即结论，正文展开 MECE 论据）
- **Visual style**: data-journalism（出版级信息密度：多栏网格、细线分隔、hero 数字、数据表格）
- **Theme**: Light theme
- **Tone**: 数据出版物般克制、专业、可信

### Color Scheme

| Role | HEX | Purpose |
| ---- | --- | ------- |
| **Background** | `#FFFFFF` | 页面背景 |
| **Secondary bg** | `#F4F7FA` | 卡片/区块背景 |
| **Primary** | `#0F2B46` | 深蓝——标题、关键区块、图标 |
| **Accent** | `#00B0F0` | 亮青——数据高亮、关键数字 |
| **Secondary accent** | `#2E7CF6` | 电蓝——第二系列、渐变过渡 |
| **Body text** | `#1A2433` | 正文 |
| **Secondary text** | `#5A6B7B` | 说明、注释 |
| **Border/divider** | `#D7E0E8` | 卡片边框、分隔线 |
| **Success** | `#1F9D55` | 入选/正向指标（绿） |
| **Warning** | `#E5533C` | 副作用/风险（红） |

### AI Image Strategy

本 deck 无 AI 生成图片（数据汇报以表格/卡片/示意图表达，保持简洁）。

---

## IV. Typography System

### Font Plan

**Typography direction**: 现代 CJK 黑体系——标题 SimHei（黑体，权威有力）领衔，正文 Microsoft YaHei（易读），数据用 Consolas 等宽增强精度感。

| Role | Chinese | English | Fallback tail |
| ---- | ------- | ------- | ------------- |
| **Title** | `SimHei` | `Arial` | sans-serif |
| **Body** | `"Microsoft YaHei", "PingFang SC"` | `Arial` | sans-serif |
| **Emphasis** | `SimHei` | `Arial` | sans-serif |
| **Code** | — | `Consolas, "Courier New"` | monospace |

**Per-role font stacks** (CSS font-family strings):

- Title: `SimHei, "Microsoft YaHei", Arial, sans-serif`
- Body: `"Microsoft YaHei", "PingFang SC", Arial, sans-serif`
- Emphasis: `SimHei, "Microsoft YaHei", Arial, sans-serif`
- Code: `Consolas, "Courier New", monospace`

### Font Size Hierarchy

**Baseline (unitless px)**: Body font size = `32`（delivery purpose: presentation）。

| Role | px | Weight |
| ---- | -- | ------ |
| Cover title | 96 | Bold |
| Page title | 56 | Bold |
| Subtitle | 44 | SemiBold |
| Lead (核心信息/每页结论) | 40 | SemiBold |
| Subheading | 36 | SemiBold |
| **Body content** | **32** | Regular |
| Annotation / caption | 24 | Regular |
| Footnote / page number | 20 | Regular |

---

## V. Layout Principles

### Page Structure

- **Header area**: 56px 高——页眉：左侧章节标签 + 右侧页码
- **Content area**: 主体 1280x560（含页眉下方 56-616 区域）
- **Footer area**: 底部 20px 注释/来源行（细线分隔）

### Layout Pattern Library

| Pattern | 使用页 | 说明 |
| ------- | ------ | ---- |
| 封面负空间（negative-space-driven） | P01 | 单一大标题 + 青蓝渐变 accent 线，留白充分 |
| 三/四列卡片 | P02 | 八维指标全景网格卡 |
| 上六下四（列表+要点） | P03 | 科学性 6 条 + 右侧结论摘要 |
| 三列卡片 + hero 数字 | P04/P05/P06 | 每列一个维度组，hero 数字领衔 |
| 单列居中（breathing） | P07 | 优化问题本质——一句核心观点 + 权衡箭头示意 |
| 四卡片 + 约束条 | P08 | 4 主目标卡 + 2 软约束横条 |
| 对称拆分（5:5） | P09 | 两组权衡关系左右对照 + 帕累托示意 |
| 全宽表格 | P10 | 指标总表 + 收尾结论 |

### Spacing Specification

**Universal**: safe margin 64px；content block gap 32px；icon-text gap 12px。
**Card-based**: card gap 28px；card padding 28px；border radius 12px。

---

## VI. Icon Usage Specification

### Source

- **Built-in icon library**: `tabler-outline`（细线专业风，stroke_width=2）

### Recommended Icon List

| Purpose | Icon Path | Page |
| ------- | --------- | ---- |
| 车辆概况 | `tabler-outline/truck` | P02 |
| 利用率 | `tabler-outline/gauge` | P02/P04 |
| 时间响应 | `tabler-outline/clock` | P02/P04 |
| 距离里程 | `tabler-outline/route` | P02/P04 |
| 货物吞吐 | `tabler-outline/package` | P02/P05 |
| 成本 | `tabler-outline/coins` | P02/P05 |
| 绿色低碳 | `tabler-outline/leaf` | P02/P06 |
| 公平 | `tabler-outline/scale` | P02/P06 |
| 可靠性 | `tabler-outline/shield-check` | P02/P06 |
| 优化对象 | `tabler-outline/target` | P08 |
| 权衡 | `tabler-outline/arrows-left-right` | P07/P09 |
| 帕累托 | `tabler-outline/chart-scatter` | P09 |
| 结论/check | `tabler-outline/circle-check` | P10 |

---

## VII. Visualization Reference List

本 deck 以表格、卡片、示意图形为主，不引用 charts 目录模板（无数据图表模板匹配页），可视化由 Executor 按 §V 布局自由绘制。少于 3 个 viz 模板页。

---

## VIII. Image Resource List

无图片资源（无 AI / 无用户图片）。

---

## IX. Content Outline

### Part 1: 指标体系与科学性

#### Slide 01 - Cover

- **Cover impact**: 数据 hook——标题即结论："TVSOS 调度系统：8 维指标体系 + 4 主目标全局优化选型"；构成：深蓝底 + 青蓝渐变 accent 线 + 大标题（typographic poster）。
- **Title**: TVSOS 统计指标体系与全局优化对象选型
- **Subtitle**: 运输车辆仿真调度系统 · 指标设计科学性 · 多目标优化方向
- **Info**: 技术汇报 · 2026-09-07

#### Slide 02 - 指标体系全景

- **Layout**: 四列卡片网格（4x2）
- **Title**: 八维指标覆盖调度全链路，估算透明、口径统一
- **Core message**: 后端统计已覆盖车辆概况/利用/时间/距离/货物/成本/绿色/公平/可靠/算法 10 个子模块。
- **Content**: 8 张维度卡（车辆概况·利用率 U·时间响应 T·距离里程·货物吞吐·成本 C·绿色 G·公平 F·可靠 R·算法 A）

#### Slide 03 - 设计科学性

- **Layout**: 左侧 6 条要点 + 右侧结论摘要
- **Title**: 指标体系设计的 6 条科学性原则
- **Core message**: 模块化、口径统一、分布优于均值、估算透明、指标驱动决策、与算法解耦。
- **Content**: 6 条科学性说明。

#### Slide 04 - 效率三支柱（U / T / D）

- **Layout**: 三列卡片 + hero 数字
- **Title**: 效率维度：装载率、空驶率与响应时间是核心
- **Core message**: U2 装载率、U4 空驶率、T2 响应 P95 构成效率三支柱，其余时间/距离指标为支撑。
- **Content**: 每列 hero 数字 + 口径 + 说明。

#### Slide 05 - 经营与可持续（C / G）

- **Layout**: 两列卡片 + 子行
- **Title**: 单位成本与单位碳排归一化，跨场景可比
- **Core message**: Cost A–F 闭环驱动调度；C2/G2 以吨公里归一，是经营与绿色核心。
- **Content**: A–F 成本链 + C2/G2 hero 数字。

#### Slide 06 - 公平与可靠（F / R）

- **Layout**: 两列卡片 + 子行
- **Title**: 基尼系数、拒单与事件指标补齐公平与可靠视角
- **Core message**: F1 基尼衡量负载均衡；R4–R6 随机事件指标量化事故/维修/休息对运力的扰动。
- **Content**: F1/F2 + R4/R5/R6 + 事件流说明。

### Part 2: 全局优化对象选型

#### Slide 07 - 优化问题本质（breathing）

- **Layout**: 单列居中，权衡箭头示意
- **Title**: 全局优化的本质：多目标权衡，不存在全指标最优单解
- **Core message**: 优化任一指标时，其他指标可能不是最优——需要帕累托视角。
- **Content**: 6 组典型权衡（U2↔T2、U4↔T2、C2↔T2、T2↔U2/U4、G2↔时效、F1↔成本）。

#### Slide 08 - 优化对象选型

- **Layout**: 四卡片（主目标）+ 两横条（软约束）
- **Title**: 4 主目标 + 2 软约束的全局优化框架
- **Core message**: C2 裁决、U2/U4/T2 过程主目标；F1/G2 软约束；其余指标监控不优化。
- **Content**: 主目标卡（C2↓ U2↑ U4↓ T2↓）+ 软约束条（F1≤0.4、G2 随 U4 改善）。

#### Slide 09 - 权衡关系

- **Layout**: 对称拆分（5:5）
- **Title**: 两组核心矛盾决定帕累托前沿
- **Core message**: U2↔T2 与 U4↔T2 构成帕累托端点；C2 是合成裁决目标。
- **Content**: 左：U2↔T2 权衡；右：U4↔T2 权衡 + C2 裁决说明。

#### Slide 10 - 指标总表与结论

- **Layout**: 全宽表格 + 底部结论
- **Title**: 结论：C2/U2/U4/T2 四目标 + F1/G2 软约束，显式管理权衡
- **Core message**: 用权重标量化 + 帕累托前沿报告显式管理指标权衡。
- **Content**: 指标总表（维度/ID/指标/是否优化对象）+ 结论行。

---

## X. Speaker Notes Requirements

每页一个备注文件存到 `notes/`，文件名与 SVG 同名（如 `01_cover.md`）。内容：结论先行（第一句即 takeaway）+ 2-3 句支撑事实，汇报口吻。
