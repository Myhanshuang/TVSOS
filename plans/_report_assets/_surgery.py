# -*- coding: utf-8 -*-
import os, re, shutil, zipfile, sys

BASE = r'D:\base\TVSOS'
WORK = os.path.join(os.environ.get('TEMP'), 'surgery')
SRC_DOCX = os.path.join(BASE, 'plans', '_report_assets', '_word_backup.docx')
OUT_DOCX = os.path.join(BASE, 'plans', '_report_assets', '_merged_report.docx')
SHOTS = os.path.join(BASE, 'plans', '_report_assets', 'shots')

# 1) 重新解包（可重复执行）
unp = os.path.join(WORK, 'unpacked')
if os.path.exists(unp): shutil.rmtree(unp)
os.makedirs(unp)
with zipfile.ZipFile(SRC_DOCX) as z: z.extractall(unp)

docp = os.path.join(unp, 'word', 'document.xml')
relp = os.path.join(unp, 'word', '_rels', 'document.xml.rels')
ctp = os.path.join(unp, '[Content_Types].xml')
doc = open(docp, encoding='utf-8').read()
rels = open(relp, encoding='utf-8').read()
ct = open(ctp, encoding='utf-8').read()

def must_replace(s, old, new, n=1):
    c = s.count(old)
    assert c == n, 'anchor count %d != %d for %r' % (c, n, old[:40])
    return s.replace(old, new)

# 2) 文本改号与引用更新（全部单 run 锚点，已探针确认 count=1）
doc = must_replace(doc, '八组卡片呈现终态。', '八组卡片呈现终态，分别如图 1-4、图 1-5 所示。')
doc = must_replace(doc, '（图 1-4）。', '（图 1-6）。')
doc = must_replace(doc, '图 1-4  基于条件最优参考答案的匹配质量评价闭环', '图 1-6  基于条件最优参考答案的匹配质量评价闭环')
doc = must_replace(doc, '（图 1-5）。', '（图 1-7）。')
doc = must_replace(doc, '图 1-5  实验沙箱流程与复现机制', '图 1-7  实验沙箱流程与复现机制')

# 3) 提取模板段落
def extract_para(anchor):
    i = doc.find(anchor)
    assert i > 0, 'anchor not found: ' + anchor[:30]
    ps = doc.rfind('<w:p ', 0, i)
    if ps < 0: ps = doc.rfind('<w:p>', 0, i)
    pe = doc.find('</w:p>', i) + len('</w:p>')
    return doc[ps:pe]

fig_tmpl = extract_para('r:embed="rId11"')
cap_tmpl = extract_para('图 1-6  基于条件最优参考答案的匹配质量评价闭环')
body_tmpl = extract_para('实时统计链路为：CalculateGlobalStats')

# 4) 段落构造函数
def hexid(n):
    return ('5A%06X' % (0xB000 + n))

def make_fig(rid, cx, cy, n):
    s = fig_tmpl
    s = s.replace('r:embed="rId11"', 'r:embed="%s"' % rid)
    s = s.replace('cx="5143500"', 'cx="%d"' % cx).replace('cy="3152775"', 'cy="%d"' % cy)
    s = s.replace('<wp:docPr id="1" name="图片 1"/>', '<wp:docPr id="%d" name="图片 %d"/>' % (n, n))
    s = s.replace('<pic:cNvPr id="1" name="图片 1"/>', '<pic:cNvPr id="%d" name="图片 %d"/>' % (n, n))
    s = re.sub(r'w14:paraId="[0-9A-F]+"', 'w14:paraId="%s"' % hexid(n), s)
    return s

def make_cap(text, n):
    # 从模板 caption 段落取 pPr，替换文字
    m = re.search(r'<w:pPr>.*?</w:pPr>', cap_tmpl, re.S)
    ppr = m.group(0)
    m2 = re.search(r'<w:r>(<w:rPr>.*?</w:rPr>)<w:t[^>]*>.*?</w:t></w:r>', cap_tmpl, re.S)
    rpr = m2.group(1)
    return ('<w:p w14:paraId="%s">%s<w:r>%s<w:t xml:space="preserve">%s</w:t></w:r></w:p>' % (hexid(n), ppr, rpr, text))

def make_body(text, n, bold_mid=None):
    m = re.search(r'<w:pPr>.*?</w:pPr>', body_tmpl, re.S)
    ppr = m.group(0)
    m2 = re.search(r'<w:r>(<w:rPr>.*?</w:rPr>)<w:t[^>]*>', body_tmpl, re.S)
    rpr = m2.group(1)
    def run(t, bold):
        rp = rpr.replace('</w:rPr>', ('<w:b/>' if bold else '') + '</w:rPr>')
        return '<w:r>%s<w:t xml:space="preserve">%s</w:t></w:r>' % (rp, t)
    if bold_mid is None:
        runs = run(text, False)
    else:
        pre, mid, post = bold_mid
        assert pre + mid + post == text
        runs = run(pre, False) + run(mid, True) + run(post, False)
    return '<w:p w14:paraId="%s">%s%s</w:p>' % (hexid(n), ppr, runs)

# 5) 五张截图（rId 从 100 起，媒体名避开 image1-7）
EMU = 9525
def ext(natw, nath, dispw=540):
    return dispw * EMU, round(nath * dispw / natw) * EMU

imgs = [
    ('rId100', 'image8.jpg',  '主界面.jpg',       (1920, 1216)),
    ('rId101', 'image9.png',  '实验报告界面.png', (2314, 1467)),
    ('rId102', 'image10.png', '沙箱管理.png',     (2308, 1460)),
    ('rId103', 'image11.png', '实验参数配置.png', (2307, 1473)),
    ('rId104', 'image12.png', '实验对比界面.png', (2316, 1465)),
]
for rid, media, src, (nw, nh) in imgs:
    shutil.copyfile(os.path.join(SHOTS, src), os.path.join(unp, 'word', 'media', media))

# 6) 注册关系与内容类型
add_rels = ''
for rid, media, src, _ in imgs:
    add_rels += '<Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/%s"/>' % (rid, media)
rels = rels.replace('</Relationships>', add_rels + '</Relationships>')
if 'Extension="jpg"' not in ct:
    ct = ct.replace('</Types>', '<Default Extension="jpg" ContentType="image/jpeg"/></Types>')

# 7) 构造插入内容
cx, cy = ext(1920, 1216)
fig_main = make_fig('rId100', cx, cy, 9001)
cx, cy = ext(2314, 1467)
fig_report = make_fig('rId101', cx, cy, 9002)
cx, cy = ext(2308, 1460)
fig_admin = make_fig('rId102', cx, cy, 9003)
cx, cy = ext(2307, 1473)
fig_config = make_fig('rId103', cx, cy, 9004)
cx, cy = ext(2316, 1465)
fig_compare = make_fig('rId104', cx, cy, 9005)

cap_main = make_cap('图 1-4  系统主界面：地图大屏与运行期数据覆盖层', 9101)
cap_report = make_cap('图 1-5  实验结果页：六项主目标卡片、SA 收敛曲线与八组终态指标', 9102)
cap_admin = make_cap('图 1-8  沙箱管理界面：世界设定卡片与实验入口', 9103)
cap_config = make_cap('图 1-9  实验参数配置界面：表单与 TOML 双向编辑', 9104)
cap_compare = make_cap('图 1-10  实验对比界面：主目标雷达与关键指标对比', 9105)

conn1_text = '沙箱管理界面与实验参数配置界面分别如图 1-8、图 1-9 所示：前者以世界设定卡片呈现各沙箱（基础设施与世界规则）及其下挂的实验方案与批次，演示沙箱内置 1146 个成都 POI 与 20 辆川A车队模板；后者提供表单与 TOML 高级编辑双向配置，涵盖随机种子、虚拟时长、时间加速比、车队规模、日均订单量与 SA 超参，并内嵌“相同种子 + 相同参数结果可复现”的口径提示。'
conn1 = make_body(conn1_text, 9201)
conn2_pre = '在复现保证之上，'
conn2_mid = '实验对比界面'
conn2_post = '（图 1-10）支持同沙箱实验的横向对比与跨沙箱追加对比：主目标雷达按“越大越优”归一化口径折算后直观呈现多批次权衡，关键指标柱状对比与结果明细辅助定位参数差异带来的影响，为多目标权衡提供受控比较的平台。'
conn2 = make_body(conn2_pre + conn2_mid + conn2_post, 9202, bold_mid=(conn2_pre, conn2_mid, conn2_post))

# 8) 执行插入
def insert_after_para(doc, anchor, payload):
    i = doc.find(anchor)
    assert i > 0, 'anchor missing: ' + anchor[:30]
    pe = doc.find('</w:p>', i) + len('</w:p>')
    return doc[:pe] + payload + doc[pe:]

block_124 = fig_main + cap_main + fig_report + cap_report
doc = insert_after_para(doc, '分别如图 1-4、图 1-5 所示。', block_124)
block_125 = conn1 + fig_admin + cap_admin + fig_config + cap_config + conn2 + fig_compare + cap_compare
doc = insert_after_para(doc, '图 1-7  实验沙箱流程与复现机制', block_125)

# 9) 写回并重打包
open(docp, 'w', encoding='utf-8').write(doc)
open(relp, 'w', encoding='utf-8').write(rels)
open(ctp, 'w', encoding='utf-8').write(ct)
if os.path.exists(OUT_DOCX): os.remove(OUT_DOCX)
zf = zipfile.ZipFile(OUT_DOCX, 'w', zipfile.ZIP_DEFLATED)
for root, dirs, files in os.walk(unp):
    for f in files:
        fp = os.path.join(root, f)
        arc = os.path.relpath(fp, unp).replace(os.sep, '/')
        zf.write(fp, arc)
zf.close()
print('MERGED OK ->', OUT_DOCX, os.path.getsize(OUT_DOCX))