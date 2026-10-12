# -*- coding: utf-8 -*-
import os, re, zipfile, shutil

DQ = chr(34)
BASE = r'D:\base\TVSOS'
WORK = os.path.join(os.environ.get('TEMP'), 'ch4')
if os.path.exists(WORK):
    shutil.rmtree(WORK)
os.makedirs(WORK)
SRC = os.path.join(BASE, 'plans', '综合设计III中期报告.docx')
unp = os.path.join(WORK, 'unpacked')
os.makedirs(unp)
with zipfile.ZipFile(SRC) as z:
    z.extractall(unp)

docp = os.path.join(unp, 'word', 'document.xml')
doc = open(docp, encoding='utf-8').read()

head_i = None
for m in re.finditer('参考文献', doc):
    ps = doc.rfind('<w:p ', 0, m.start())
    pe = doc.find('</w:p>', m.start()) + len('</w:p>')
    para = doc[ps:pe]
    if 'pageBreakBefore' in para and ('w:val=' + DQ + '2' + DQ) in para:
        head_i = (ps, pe, para)
        break
assert head_i, 'references heading not found'
H1_TMPL = head_i[2]
H1_TMPL = re.sub(r'<w:bookmark(Start|End)[^>]*/>', '', H1_TMPL)
H1_TMPL = re.sub(r'w14:paraId=' + DQ + '[0-9A-F]+' + DQ, 'w14:paraId=' + DQ + '5B40001' + DQ, H1_TMPL)
TITLE = '第四章  AI工具利用情况与思考'
tmatch = re.search(r'<w:t[^>]*>参考文献</w:t>', H1_TMPL)
assert tmatch, 'heading text run not found'
H1_XML = H1_TMPL[:tmatch.start()] + tmatch.group(0).replace('参考文献', TITLE) + H1_TMPL[tmatch.end():]

bi = doc.find('实时统计链路为：')
assert bi > 0
b_ps = doc.rfind('<w:p ', 0, bi)
b_pe = doc.find('</w:p>', bi) + len('</w:p>')
BODY_TMPL = doc[b_ps:b_pe]
b_ppr = re.search(r'<w:pPr>.*?</w:pPr>', BODY_TMPL, re.S).group(0)
b_rpr = re.search(r'<w:r>(<w:rPr>.*?</w:rPr>)<w:t[^>]*>', BODY_TMPL, re.S).group(1)

qi = doc.find('（1）货物材料')
assert qi > 0, 'bold lead text not found'
r_ps = doc.rfind('<w:r>', 0, qi)
r_pe = doc.find('</w:r>', qi) + len('</w:r>')
bold_run = doc[r_ps:r_pe]
bold_rpr = re.search(r'<w:rPr>.*?</w:rPr>', bold_run, re.S).group(0)
assert '<w:b/>' in bold_rpr, 'bold rPr has no b tag'

def make_para(nid, runs):
    out = '<w:p w14:paraId=' + DQ + '5B40' + nid + DQ + '>' + b_ppr
    for txt, bold in runs:
        rp = bold_rpr if bold else b_rpr
        out += '<w:r>' + rp + '<w:t xml:space=' + DQ + 'preserve' + DQ + '>' + txt + '</w:t></w:r>'
    return out + '</w:p>'

import json
content = json.load(open(os.path.join(os.path.dirname(os.path.abspath(__file__)), '_ch4_content.json'), encoding='utf-8'))['items']

block = H1_XML
for idx, item in enumerate(content):
    nid = '%03d' % (idx + 2)
    runs = [(item['q'], True), (item['a'], False)] if 'q' in item else [(item['t'], False)]
    block += make_para(nid, runs)

doc = doc[:head_i[0]] + block + doc[head_i[0]:]
open(docp, 'w', encoding='utf-8').write(doc)

OUT = os.path.join(WORK, 'ch4_step1.docx')
zf = zipfile.ZipFile(OUT, 'w', zipfile.ZIP_DEFLATED)
for root, dirs, files in os.walk(unp):
    for f in files:
        fp = os.path.join(root, f)
        arc = os.path.relpath(fp, unp).replace(os.sep, '/')
        zf.write(fp, arc)
zf.close()
print('STEP1 OK:', OUT, os.path.getsize(OUT))