# -*- coding: utf-8 -*-
import os, re, zipfile

WORK = os.path.join(os.environ.get('TEMP'), 'tocfix')
unp = os.path.join(WORK, 'unpacked')
OUT = r'D:\base\TVSOS\plans\_report_assets\_styles_fixed.docx'

stp = os.path.join(unp, 'word', 'styles.xml')
sty = open(stp, encoding='utf-8').read()

# 1) 每个样式内：uiPriority 移到 schema 合法位置（semiHidden/unhideWhenUsed/qFormat/pPr/rPr 之前）
LATER = ['<w:semiHidden', '<w:unhideWhenUsed', '<w:qFormat', '<w:locked', '<w:personal', '<w:pPr', '<w:rPr', '<w:tblPr', '</w:style>']
fixed_ui = [0]
def fix_style_seg(seg):
    uis = re.findall(r'<w:uiPriority w:val="(\d+)"\s*/>', seg)
    if not uis:
        return seg
    seg2 = re.sub(r'<w:uiPriority w:val="\d+"\s*/>', '', seg)
    val = uis[0]
    pos = None
    for mk in LATER:
        p = seg2.find(mk)
        if p > 0:
            pos = p; break
    if pos is None:
        return seg
    fixed_ui[0] += 1
    return seg2[:pos] + '<w:uiPriority w:val="%s"/>' % val + seg2[pos:]

def repl_style(m):
    return fix_style_seg(m.group(0))

sty2 = re.sub(r'<w:style [^>]*>.*?</w:style>', repl_style, sty, flags=re.S)
print('uiPriority repositioned:', fixed_ui[0])

# 2) 标题样式补 outlineLvl（styleId 2..7 = heading 1..6 → 级别 0..5）
added = []
for sid in ['2','3','4','5','6','7']:
    lvl = int(sid) - 2
    m = re.search(r'<w:style ([^>]*w:styleId="%s"[^>]*)>(.*?)</w:style>' % sid, sty2, flags=re.S)
    if not m:
        print('style %s not found' % sid); continue
    seg = m.group(0)
    if '<w:outlineLvl' in seg:
        print('style %s already has outlineLvl' % sid); continue
    nm = re.search(r'<w:name w:val="([^"]+)"', seg)
    if not nm or 'heading' not in nm.group(1):
        print('style %s name=%s - skip' % (sid, nm.group(1) if nm else '?')); continue
    if '<w:pPr>' in seg:
        seg2 = seg.replace('</w:pPr>', '<w:outlineLvl w:val="%d"/></w:pPr>' % lvl, 1)
    else:
        rp = seg.find('<w:rPr')
        if rp > 0:
            seg2 = seg[:rp] + '<w:pPr><w:outlineLvl w:val="%d"/></w:pPr>' % lvl + seg[rp:]
        else:
            seg2 = seg.replace('</w:style>', '<w:pPr><w:outlineLvl w:val="%d"/></w:pPr></w:style>' % lvl, 1)
    sty2 = sty2.replace(seg, seg2, 1)
    added.append(sid)
print('outlineLvl added to styles:', added)

open(stp, 'w', encoding='utf-8').write(sty2)

# 3) 重打包为 _styles_fixed.docx 供 COM 更新
if os.path.exists(OUT): os.remove(OUT)
zf = zipfile.ZipFile(OUT, 'w', zipfile.ZIP_DEFLATED)
for root, dirs, files in os.walk(unp):
    for f in files:
        fp = os.path.join(root, f)
        arc = os.path.relpath(fp, unp).replace(os.sep, '/')
        zf.write(fp, arc)
zf.close()
print('PACKED:', OUT, os.path.getsize(OUT))