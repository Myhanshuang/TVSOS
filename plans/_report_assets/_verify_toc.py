# -*- coding: utf-8 -*-
import os, re
WORK = os.path.join(os.environ.get('TEMP'), 'tocfix')
unp = os.path.join(WORK, 'unpacked')
BASE = r'D:\base\TVSOS'
COMOUT = os.path.join(BASE, 'plans', '_report_assets', '_com_toc_out.txt')
DQ = chr(34)
BS = chr(92)

doc = open(os.path.join(unp, 'word', 'document.xml'), encoding='utf-8').read()
SEP = 'fldCharType=' + DQ + 'separate' + DQ
END = 'fldCharType=' + DQ + 'end' + DQ

def norm(s):
    return re.sub(r'\s+', '', s)

expect = []
for line in open(COMOUT, encoding='utf-8', errors='replace').read().splitlines():
    parts = line.split('|')
    if len(parts) == 3 and parts[0] == 'TOCPARA':
        flat = parts[2].strip('~').strip()
        if not flat:
            continue
        segs = flat.split('~')
        if len(segs) >= 2 and segs[-1].strip().isdigit():
            expect.append((norm(''.join(segs[:-1])), segs[-1].strip()))

ok = 0
bad = []
for n in range(1, 18):
    pk = 'PAGEREF _Toc%d ' % n + BS + 'h'
    pi = doc.find(pk)
    assert pi > 0, 'pageref missing _Toc%d' % n
    pj = doc.find(END, pi)
    nums = re.findall(r'<w:t[^>]*>([^<]*)</w:t>', doc[pi:pj])
    num = nums[-1] if nums else '(EMPTY)'
    tab = doc.rfind('<w:tab/>', 0, pi)
    prevsep = doc.rfind(SEP, 0, tab)
    title = norm(''.join(re.findall(r'<w:t[^>]*>([^<]*)</w:t>', doc[prevsep:tab])))
    exp_t, exp_n = expect[n-1]
    match = (title == exp_t and num == exp_n)
    if match:
        ok += 1
    else:
        bad.append((n, title, num, exp_t, exp_n))
    print('%02d %s -> %s expect %s %s' % (n, title[:22], num, exp_n, 'OK' if match else 'MISMATCH'))

print('MATCHED:', ok, 'of 17')
if bad:
    print('BAD:', bad)