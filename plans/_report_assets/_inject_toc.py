# -*- coding: utf-8 -*-
import os, re, zipfile

BASE = r'D:\base\TVSOS'
WORK = os.path.join(os.environ.get('TEMP'), 'tocfix')
unp = os.path.join(WORK, 'unpacked')
COMOUT = os.path.join(BASE, 'plans', '_report_assets', '_com_toc_out.txt')
OUT = os.path.join(BASE, 'plans', '_report_assets', '_toc_fixed.docx')
DQ = chr(34)
BS = chr(92)

def norm(s):
    return re.sub(r'\s+', '', s)

title2num = {}
order = []
for line in open(COMOUT, encoding='utf-8', errors='replace').read().splitlines():
    parts = line.split('|')
    if len(parts) == 3 and parts[0] == 'TOCPARA':
        flat = parts[2].strip('~').strip()
        if not flat:
            continue
        segs = flat.split('~')
        if len(segs) >= 2 and segs[-1].strip().isdigit():
            title = norm(''.join(segs[:-1]))
            title2num[title] = segs[-1].strip()
            order.append((title, segs[-1].strip()))
print('parsed entries:', len(title2num))
assert len(title2num) == 17, 'expect 17 TOC entries'

docp = os.path.join(unp, 'word', 'document.xml')
doc = open(docp, encoding='utf-8').read()
SEP = 'fldCharType=' + DQ + 'separate' + DQ
END = 'fldCharType=' + DQ + 'end' + DQ

def entry_title_for(n):
    key = 'HYPERLINK ' + BS + 'l "_Toc%d"' % n
    i = doc.find(key)
    if i < 0:
        return None
    sep = doc.find(SEP, i)
    tab = doc.find('<w:tab/>', sep)
    span = doc[sep:tab]
    texts = re.findall(r'<w:t[^>]*>([^<]*)</w:t>', span)
    return norm(''.join(texts))

injected = 0
fallbacks = 0
for n in range(1, 18):
    t = entry_title_for(n)
    num = title2num.get(t) if t else None
    if num is None:
        idx = n - 1
        if idx < len(order):
            num = order[idx][1]
            fallbacks += 1
    assert num, 'no number for _Toc%d' % n
    key = 'PAGEREF _Toc%d ' % n + BS + 'h'
    cnt = doc.count(key)
    assert cnt == 1, 'PAGEREF key count %d for _Toc%d' % (cnt, n)
    i = doc.find(key)
    j = doc.find(END, i)
    rs = doc.rfind('<w:r>', i, j)
    assert rs > i, 'end-run not found for _Toc%d' % n
    run = '<w:r><w:t>%s</w:t></w:r>' % num
    doc = doc[:rs] + run + doc[rs:]
    injected += 1
print('injected:', injected, 'order-fallback used:', fallbacks)
open(docp, 'w', encoding='utf-8').write(doc)

stp = os.path.join(unp, 'word', 'settings.xml')
st = open(stp, encoding='utf-8').read()
if 'updateFields' not in st:
    UF = '<w:updateFields w:val=' + DQ + 'true' + DQ + '/>'
    placed = False
    for anchor in ['<w:hdrShapeDefaults', '<w:footnotePr', '<w:endnotePr', '<w:compat', '<w:docVars', '<w:rsids', '</w:settings>']:
        if anchor in st:
            st = st.replace(anchor, UF + anchor, 1)
            print('updateFields inserted before:', anchor)
            placed = True
            break
    if not placed:
        print('WARN: updateFields not placed')
else:
    print('updateFields already present')
open(stp, 'w', encoding='utf-8').write(st)

if os.path.exists(OUT):
    os.remove(OUT)
zf = zipfile.ZipFile(OUT, 'w', zipfile.ZIP_DEFLATED)
for root, dirs, files in os.walk(unp):
    for f in files:
        fp = os.path.join(root, f)
        arc = os.path.relpath(fp, unp).replace(os.sep, '/')
        zf.write(fp, arc)
zf.close()
print('PACKED:', OUT, os.path.getsize(OUT))