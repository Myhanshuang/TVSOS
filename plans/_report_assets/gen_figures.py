import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
from matplotlib.patches import FancyBboxPatch, FancyArrowPatch
import os

plt.rcParams['font.sans-serif'] = ['Microsoft YaHei', 'SimHei']
plt.rcParams['axes.unicode_minus'] = False
OUT = r'D:\base\TVSOS\plans\_report_assets'
os.makedirs(OUT, exist_ok=True)

def box(ax, x, y, w, h, text, fc='#E8F1FA', ec='#2F5D8C', fs=10.5, bold=False, tc='#1A2B3C'):
    p = FancyBboxPatch((x, y), w, h, boxstyle='round,pad=0.6,rounding_size=1.2', fc=fc, ec=ec, lw=1.3)
    ax.add_patch(p)
    weight = 'bold' if bold else 'normal'
    ax.text(x+w/2, y+h/2, text, ha='center', va='center', fontsize=fs, weight=weight, color=tc, linespacing=1.45)

def arrow(ax, x1, y1, x2, y2, color='#3A4A5A', lw=1.6, style='-|>', ms=14, ls='-'):
    a = FancyArrowPatch((x1, y1), (x2, y2), arrowstyle=style, mutation_scale=ms, color=color, lw=lw, linestyle=ls)
    ax.add_patch(a)

def arcarrow(ax, x1, y1, x2, y2, rad, color='#3A4A5A', lw=1.5, ls='-'):
    a = FancyArrowPatch((x1, y1), (x2, y2), arrowstyle='-|>', mutation_scale=13, color=color, lw=lw,
                        linestyle=ls, connectionstyle='arc3,rad=%s' % rad)
    ax.add_patch(a)

def label(ax, x, y, text, fs=9, color='#444444', ha='center', weight='normal'):
    ax.text(x, y, text, ha=ha, va='center', fontsize=fs, color=color, weight=weight, linespacing=1.4)

def newfig(w, h):
    fig, ax = plt.subplots(figsize=(w, h))
    ax.set_xlim(0, 100); ax.set_ylim(0, 100)
    ax.axis('off')
    return fig, ax

def save(fig, name):
    fig.savefig(os.path.join(OUT, name), dpi=150, bbox_inches='tight', facecolor='white')
    plt.close(fig)
    print('saved', name)

# ---------------- 图1 系统总体架构 ----------------
fig, ax = newfig(11.5, 7.0)
box(ax, 1, 84, 98, 14, '', fc='#F3EDF9', ec='#7A5FA0')
label(ax, 50, 95.5, '展示层（Vue3 + Element Plus + ECharts + 高德地图）', fs=12, weight='bold', color='#4A3A66')
for t, x in [('沙箱管理', 3.5), ('实验配置\n表单/TOML双向', 20), ('地图大屏\n数据覆盖层', 36.5), ('任务管理', 53), ('实验结果\n八组卡片', 69.5), ('实验对比\n分栏Diff', 86)]:
    box(ax, x, 85.8, 11, 8.4, t, fc='white', ec='#7A5FA0', fs=9.5)
box(ax, 1, 63, 98, 14, '', fc='#E8F1FA', ec='#2F5D8C')
label(ax, 50, 74.5, '服务层（Go + Gin）', fs=12, weight='bold', color='#1F3D5C')
box(ax, 6, 64.8, 38, 8.4, 'REST API（登录鉴权 AuthRequired\n中间件 + 业务路由）', fc='white', ec='#2F5D8C', fs=9.5)
box(ax, 56, 64.8, 38, 8.4, 'WebSocket 实时推送\n（/ws/stats 每 3s 快照，前端 15s 轮询兜底）', fc='white', ec='#2F5D8C', fs=9.5)
box(ax, 1, 36, 98, 24, '', fc='#EAF7EE', ec='#3B7A4E')
label(ax, 50, 57, '仿真运行层（ExperimentRuntime：单实验串行 · 单线程确定性 tick）', fs=12, weight='bold', color='#275C37')
label(ax, 50, 52.5, 'masterLoop 每个 tick 固定顺序：释放订单 → SA 派单 → 消费事件脚本 → 恢复（维修/休息） → 移动 → 指标快照', fs=9.5, color='#2F5D3B')
for t, x, w in [('虚拟时钟\n起点+tick×步长', 3.5, 17), ('SA 模拟退火调度\n多目标 Cost A–F', 21, 17), ('随机事件引擎\n拒单/疲劳/天气/车祸', 42, 17), ('全局统计引擎\n八维指标实时计算', 66, 17), ('种子 RNG\nrt.rng 严格复现', 87.5, 11)]:
    box(ax, x, 37.6, w, 9.5, t, fc='white', ec='#3B7A4E', fs=9)
box(ax, 1, 12, 98, 20, '', fc='#FDF3E7', ec='#B07A2A')
label(ax, 50, 29.5, '数据与生成层', fs=12, weight='bold', color='#7A5318')
box(ax, 4, 13.8, 42, 10.5, '确定性数据生成器（internal/sandbox/gen）\n种子 → 订单流(泊松/均匀) + 事件脚本 + 司机等级\n同种子同参数 → 输出逐字节一致', fc='white', ec='#B07A2A', fs=9)
box(ax, 53, 13.8, 43, 10.5, 'MySQL（GORM）\n沙箱元数据 5 表 + 运行表 9 张（experiment_id 隔离）\n指标快照 / 匹配日志 / 事件与轨迹记录', fc='white', ec='#B07A2A', fs=9)
for x in (22, 50, 78):
    arrow(ax, x, 84, x, 77.6)
    arrow(ax, x, 63, x, 60.2)
    arrow(ax, x, 36, x, 32.5)
save(fig, 'fig1_arch.png')

# ---------------- 图2 司机行为与随机事件状态机（重排：四带无交叉） ----------------
fig, ax = newfig(11.5, 7.4)
# 带1：司机行为 + 概率参数表
box(ax, 1, 77, 98, 21, '', fc='#F3EDF9', ec='#7A5FA0')
label(ax, 50, 95.3, '司机行为状态与事件概率参数（数学概率基准，配置可调）', fs=11.5, weight='bold', color='#4A3A66')
box(ax, 4, 85, 9, 8, '司机可用', fc='white', ec='#7A5FA0', fs=10, bold=True)
box(ax, 22, 85, 12, 8, '休息中\n(rest_until)', fc='white', ec='#7A5FA0', fs=9)
arrow(ax, 13.3, 90, 21.7, 90, color='#7A5FA0')
label(ax, 18, 94.3, '连续驾驶≥4h → 强制休息20min', fs=8, color='#8A4B00')
arrow(ax, 24, 83.2, 13.5, 83.2, color='#999999', ls='--')
label(ax, 19.5, 81.2, '休息到期自动恢复', fs=8, color='#555555')
arcarrow(ax, 6.5, 84.3, 10.5, 84.3, -0.5, color='#C25B4E')
label(ax, 8, 80.4, '拒单 p=0.06×等级修正', fs=7.6, color='#8A4B00')
label(ax, 8, 78.2, '同单≤2次·运单回池重派', fs=7.6, color='#8A4B00')
box(ax, 52, 79, 46, 16.5, '', fc='white', ec='#7A5FA0')
ax.text(54, 93.3, '事件概率与参数（均写入配置 / 双层 TOML）', ha='left', va='center', fontsize=9.5, weight='bold', color='#4A3A66')
for yy, txt in [(90.4, '· 拒单：基准 p=0.06 × 等级修正因子（1.4→0.7），同单≤2次'),
                (87.7, '· 疲劳：连续驾驶≥4h → 休息 20min（法规阈值）'),
                (85.0, '· 极端天气：p=0.0005/tick → 暂停 0.5~2h'),
                (82.3, '· 车祸：p=0.0002/tick × 等级因子 → 维修 1~6h')]:
    ax.text(54.5, yy, txt, ha='left', va='center', fontsize=9, color='#333333')
# 带2：运输随机事件状态机（两行对称）
box(ax, 1, 47, 98, 27, '', fc='#EAF7EE', ec='#3B7A4E')
label(ax, 50, 71.3, '运输过程随机事件状态机（预生成脚本到点消费）', fs=11.5, weight='bold', color='#275C37')
box(ax, 2, 56, 10, 8, '行驶中', fc='white', ec='#3B7A4E', fs=10, bold=True)
box(ax, 20, 56, 20, 8, '天气暂停（封路/限行）\n暂停 0.5~2h', fc='white', ec='#3B7A4E', fs=8.8)
arrow(ax, 12.3, 61, 19.7, 61, color='#3B7A4E')
label(ax, 16, 65.5, '极端天气 p=0.0005/tick', fs=8, color='#8A4B00')
arrow(ax, 24, 53.8, 13.5, 53.8, color='#999999', ls='--')
label(ax, 18.5, 51.5, '天气窗口结束自动恢复', fs=8, color='#555555')
box(ax, 54, 56, 10, 8, '行驶中', fc='white', ec='#3B7A4E', fs=10, bold=True)
box(ax, 72, 56, 22, 8, '维修中（车辆状态=4）\n维修 1~6h · 在途任务挂起', fc='white', ec='#3B7A4E', fs=8.8)
arrow(ax, 64.3, 61, 71.7, 61, color='#C25B4E')
label(ax, 68, 65.5, '车祸 p=0.0002/tick × 等级因子', fs=8, color='#8A4B00')
arrow(ax, 76, 53.8, 65.5, 53.8, color='#999999', ls='--')
label(ax, 70.5, 51.5, '修复完成 → 空闲（有任务则续跑）', fs=8, color='#555555')
# 带3：统一暂停机制流水线
box(ax, 1, 20, 98, 24, '', fc='#E8F1FA', ec='#2F5D8C')
label(ax, 50, 41.3, '统一暂停机制：moveVehicle 每 tick 判定（虚拟时间比较）', fs=11.5, weight='bold', color='#1F3D5C')
steps = [('派单\n(Planned)', 3.5), ('发车前\n拒单判定', 20), ('装货→在途\n轨迹推进', 36.5), ('天气/车祸\n脚本到点消费', 53), ('暂停窗口\npause_until', 70), ('恢复行驶/\n修复为空闲', 86.5)]
for t, x in steps:
    box(ax, x, 24, 11, 10, t, fc='white', ec='#2F5D8C', fs=8.8)
for i in range(5):
    arrow(ax, steps[i][1]+11.2, 29, steps[i+1][1]-0.3, 29, color='#2F5D8C')
label(ax, 50, 21.3, 'pause_until 未到期 → 本 tick 车辆位置冻结、速度为 0；休息/天气/维修三类延后共用同一检查点', fs=9, color='#1F3D5C')
# 带4：留痕与指标联动
box(ax, 1, 1, 98, 15, '', fc='#FDF3E7', ec='#B07A2A')
label(ax, 50, 13.5, '事件留痕与指标联动', fs=11, weight='bold', color='#7A5318')
box(ax, 4, 2, 27, 8.5, 'random_event 事件表\n类型/车辆/司机/运单/严重度/时长', fc='white', ec='#B07A2A', fs=8.6)
box(ax, 36.5, 2, 27, 8.5, '前端事件流时间线\n（车牌展示·类型着色·未解决空心）', fc='white', ec='#B07A2A', fs=8.6)
box(ax, 69, 2, 27, 8.5, 'R 系可靠性指标\nMTBF/MTTR/可用性 + 事件计数', fc='white', ec='#B07A2A', fs=8.6)
arrow(ax, 24, 19.5, 17, 11.2, color='#B07A2A')
arrow(ax, 50, 19.5, 50, 11.2, color='#B07A2A')
arrow(ax, 76, 19.5, 83, 11.2, color='#B07A2A')
save(fig, 'fig2_events.png')

# ---------------- 图3 匹配质量评价闭环 ----------------
fig, ax = newfig(11.5, 5.6)
label(ax, 50, 95, '基于条件最优参考答案的匹配质量评价闭环', fs=12, weight='bold', color='#1A2B3C')
box(ax, 2, 62, 20, 24, 'SA 批量派单结果\n（运单→车辆指派\n+ 整批综合 Cost）', fc='#E8F1FA', ec='#2F5D8C', fs=9.5)
box(ax, 26, 62, 24, 24, '逐单条件枚举\n固定其余指派不变，\n将该单改派各候选车，\n计算整批成本 O(单×车)', fc='#EAF7EE', ec='#3B7A4E', fs=9.5)
box(ax, 54, 62, 20, 24, '成本升序候选排名\n（Top8 JSON 回填\nmatching_log 供复盘）', fc='#EAF7EE', ec='#3B7A4E', fs=9.5)
box(ax, 78, 62, 20, 24, '对照判定\n选中车 = 排名首位？\n（条件最优车）', fc='#EAF7EE', ec='#3B7A4E', fs=9.5)
arrow(ax, 22.3, 74, 25.8, 74)
arrow(ax, 50.3, 74, 53.8, 74)
arrow(ax, 74.3, 74, 77.8, 74)
box(ax, 2, 20, 47, 26, 'M1 精确率 = 选中恰为条件最优的占比\nM2 召回率 = 分母含未派单运单\nM3 F1 = 2PR/(P+R)\nM4 MRR = 选中车排名倒数的均值', fc='#FDF3E7', ec='#B07A2A', fs=10)
box(ax, 53, 20, 45, 26, 'C1 基线对比：同批次重放贪心首可行策略\nC1 = (基线成本 − SA成本) / 基线成本 × 100%\n+ A2 求解耗时（RecordSolverTiming）\n+ A3 精确枚举 gap（≤4单≤6车时）', fc='#FDF3E7', ec='#B07A2A', fs=10)
arrow(ax, 88, 61.5, 88, 46.5, color='#B07A2A')
arrow(ax, 75.5, 33, 53, 33, color='#B07A2A', style='<|-|>')
label(ax, 50, 9, '评价结果每 3s 随 RealtimeSummary 推送前端（底板最终指标值），实验态按窗口累计并落结果页', fs=9.5, color='#555555')
save(fig, 'fig3_quality.png')

# ---------------- 图4 优化目标与指标体系 ----------------
fig, ax = newfig(11.5, 6.6)
label(ax, 50, 96, '全局优化对象：4 项主目标 + 2 项软约束', fs=12, weight='bold', color='#8A1F1F')
for i, (t, fc_, ec_) in enumerate([('C2 单位成本 ↓', '#FBE9E7', '#C25B4E'), ('U2 载重利用率 ↑', '#FBE9E7', '#C25B4E'), ('U4 空驶率 ↓', '#FBE9E7', '#C25B4E'), ('T2 响应 P95 ↓', '#FBE9E7', '#C25B4E'), ('F1 负载基尼 ≤ 0.4（软）', '#FFF8E1', '#B07A2A'), ('G2 吨公里碳排（软）', '#FFF8E1', '#B07A2A')]):
    box(ax, 2.5 + i*16.5, 82, 14.5, 10, t, fc=fc_, ec=ec_, fs=9.3)
label(ax, 50, 76, '↑ 由八维指标体系支撑 ↓', fs=9.5, color='#555555')
for t, x in [('M 匹配质量\nP/R/F1/MRR', 2.5), ('U 运力利用\n装载率/空驶率', 16.2), ('T 时效响应\nP50/P95/P99', 29.9), ('A 算法性能\n耗时/枚举gap/改善', 43.6), ('C 成本\n总成本/单位成本', 57.3), ('R 可靠性\nMTBF/MTTR/可用性', 71), ('F 公平性\n基尼系数/长尾', 84.7)]:
    box(ax, x, 52, 12.8, 17, t, fc='#E8F1FA', ec='#2F5D8C', fs=8.8)
label(ax, 50, 44.5, '实时统计链路', fs=11.5, weight='bold', color='#1F3D5C')
box(ax, 2, 22, 92, 16, '', fc='#EAF7EE', ec='#3B7A4E')
box(ax, 4.5, 24, 26, 12, 'CalculateGlobalStats\n（每虚拟 tick 全量计算）', fc='white', ec='#3B7A4E', fs=9.3)
box(ax, 35, 24, 26, 12, 'BuildRealtimeDashboard\n→ RealtimeSummary', fc='white', ec='#3B7A4E', fs=9.3)
box(ax, 65.5, 24, 26, 12, 'WS 推送 3s + 15s 轮询兜底\n每 6 tick 落 metric_snapshot', fc='white', ec='#3B7A4E', fs=9.3)
arrow(ax, 30.8, 30, 34.8, 30)
arrow(ax, 61.3, 30, 65.3, 30)
arrow(ax, 50, 51.5, 50, 38.5)
arrow(ax, 50, 79.5, 50, 69.5)
label(ax, 50, 13, '前端：地图数据覆盖层（运行健康/运力效率/Top3最值/五维雷达/最终指标值）；实验结果页按 M/U/T/A/C/R/G+事件 八组卡片呈现', fs=9, color='#555555')
save(fig, 'fig4_metrics.png')

# ---------------- 图5 实验沙箱流程与复现机制 ----------------
fig, ax = newfig(11.5, 6.0)
label(ax, 50, 95.5, '实验沙箱：受控实验与严格复现', fs=12, weight='bold', color='#1A2B3C')
box(ax, 2, 74, 20, 15, '沙箱 Sandbox\n世界设定\n(POI/车队/货类白名单)', fc='#F3EDF9', ec='#7A5FA0', fs=8.8)
box(ax, 26, 74, 20, 15, '实验方案 Scheme\n参数模板\n(双层 TOML)', fc='#F3EDF9', ec='#7A5FA0', fs=8.8)
box(ax, 50, 74, 20, 15, '新建实验 Experiment\n参数快照 + 种子\n(表单/TOML 双向编辑)', fc='#F3EDF9', ec='#7A5FA0', fs=8.8)
box(ax, 74, 74, 24, 15, '确定性数据生成器\n订单流 + 事件脚本 + 司机等级\nDataGeneratorVersion v2 校验', fc='#EAF7EE', ec='#3B7A4E', fs=8.8)
arrow(ax, 22.3, 81.5, 25.8, 81.5)
arrow(ax, 46.3, 81.5, 49.8, 81.5)
arrow(ax, 70.3, 81.5, 73.8, 81.5)
box(ax, 2, 46, 30, 20, '开始实验\nmasterLoop 单线程 tick\n虚拟时钟：时间=起点+tick×步长\n倍速只改真实节奏', fc='#E8F1FA', ec='#2F5D8C', fs=9.3)
box(ax, 36, 46, 28, 20, '运行观测\n实时指标推送 + 每 6 tick 快照\n9 张运行表按 experiment_id 隔离', fc='#E8F1FA', ec='#2F5D8C', fs=9.3)
box(ax, 68, 46, 30, 20, '到达固定虚拟时长自动完成\n结果 = summary 全量\n+ 6 项主目标 + SA 收敛曲线', fc='#E8F1FA', ec='#2F5D8C', fs=9.3)
arrow(ax, 86, 73.5, 82, 67, color='#3B7A4E')
arrow(ax, 32.3, 56, 35.8, 56)
arrow(ax, 64.3, 56, 67.8, 56)
box(ax, 2, 18, 45, 20, '对比页：参数 TOML 分栏 Diff\n（LCS 行级差异，红删绿增）', fc='#FDF3E7', ec='#B07A2A', fs=9.5)
box(ax, 53, 18, 45, 20, '导出报告：HTML / Markdown / JSON\n（主目标 + 分组全景 + 事件流 + 收敛曲线）', fc='#FDF3E7', ec='#B07A2A', fs=9.5)
arrow(ax, 83, 45.5, 30, 39.5, color='#B07A2A')
arrow(ax, 83, 45.5, 75, 39.5, color='#B07A2A')
box(ax, 2, 2, 96, 11, '复现性验证：同种子 + 同参数 + 同生成器版本 → 两次实验总成本逐位一致（35025.80213604554）；种子可导出/导入跨机重放', fc='#FFF8E1', ec='#B07A2A', fs=9.5)
arrow(ax, 75, 17.5, 62, 13.5, color='#B07A2A', ls='--')
save(fig, 'fig5_sandbox.png')

# ---------------- 图6 需求发生链条方案（重排：标注不压框） ----------------
fig, ax = newfig(11.0, 4.8)
label(ax, 50, 93, '货物材料需求发生链条设计（后续实现方案）', fs=12, weight='bold', color='#1A2B3C')
chain = [('原木\n(原材料)\n吨位 · 方量', 4, '#F3EDF9'), ('木材\n(一次加工)\n吨位 · 方量', 27, '#E8F1FA'), ('板材\n(二次加工)\n吨位 · 方量', 50, '#EAF7EE'), ('家具\n(终端产品)\n吨位 · 方量', 73, '#FDF3E7')]
for t, x, fc_ in chain:
    box(ax, x, 68, 18, 16, t, fc=fc_, ec='#2F5D8C', fs=10, bold=True)
for x1 in (22, 45, 68):
    arrow(ax, x1+0.5, 76, x1+4.3, 76, lw=2.0)
label(ax, 24.5, 63.8, '加工产出率 α（损耗折算）', fs=8.5, color='#8A4B00')
label(ax, 47.5, 63.8, '加工产出率 α（损耗折算）', fs=8.5, color='#8A4B00')
label(ax, 70.5, 63.8, '加工产出率 α（损耗折算）', fs=8.5, color='#8A4B00')
arrow(ax, 13, 67.3, 13, 42.8, color='#C25B4E')
arrow(ax, 82, 67.3, 82, 42.8, color='#C25B4E')
box(ax, 4, 20, 87, 22, '', fc='#FBE9E7', ec='#C25B4E')
label(ax, 47.5, 38.5, '传导规则（拟在数据生成器实现）', fs=10.5, weight='bold', color='#7A2E24')
for yy, txt in [(34.3, '① 各节点基础需求按泊松过程独立到达（保留现有订单流模型）'),
                (30.1, '② 上游节点订单成交后，以概率 β 经加工延迟 L 触发下游节点补货订单'),
                (25.9, '③ 下游订单吨位 = 上游吨位 × 产出率 α，方量按折算系数换算（体现加工损耗趋势）'),
                (23.0, '④ 链条拓扑与 α/β/L 参数写入世界规则 TOML，生成器版本升 v3，保持逐字节可复现')]:
    label(ax, 47.5, yy, txt, fs=9, color='#5A2E24')
save(fig, 'fig6_chain.png')

print('ALL DONE')