# -*- coding: utf-8 -*-
import json, re
BASE = r'D:\base\TVSOS'
js_path = BASE + r'\plans\_report_assets\docx_ch23.js'
items = json.load(open(BASE + r'\plans\_report_assets\_ch4_content.json', encoding='utf-8'))['items']
js = open(js_path, encoding='utf-8').read()
assert '第四章' not in js, 'chapter4 already in source'

def esc(s):
    return s.replace(chr(92), chr(92)*2).replace(chr(34), chr(92)+chr(34))

lines = []
lines.append('// ================= chapter 4 =================')
lines.append('var ch4 = [];')
lines.append('ch4.push(H.H1(' + chr(34) + '第四章  AI工具利用情况与思考' + chr(34) + '));')
for item in items:
    if 'q' in item:
        lines.append('ch4.push(H.P([H.b(' + chr(34) + esc(item['q']) + chr(34) + '), H.t(' + chr(34) + esc(item['a']) + chr(34) + ')]));')
    else:
        lines.append('ch4.push(H.P(' + chr(34) + esc(item['t']) + chr(34) + '));')
block = chr(10).join(lines) + chr(10)

marker = '// ================= references ================='
assert marker in js
js = js.replace(marker, block + marker, 1)
js = js.replace('out = ch2.concat(ch3, refs);', 'out = ch2.concat(ch3, ch4, refs);')
open(js_path, 'w', encoding='utf-8').write(js)
print('SOURCE SYNCED, ch4 lines:', len(lines))