# 用 (GameReceiver::Server*) 诊断串反查 LEA 定位全部 Server* 函数真身
# （壳内重排导致旧文档地址系失准，以字符串 xref 为准重建）
import struct, re, sys
from capstone import Cs, CS_ARCH_X86, CS_MODE_64

DLL = r'G:/pojie/ai/9yin-server-go/artifacts/fxgame/fxnet2_x64.dll'
data = open(DLL, 'rb').read()
TEXT_VA, TEXT_SZ, TEXT_RAW = 0x1000, 0x3ccea, 0x400

md = Cs(CS_ARCH_X86, CS_MODE_64)

def rva2off(rva):
    if TEXT_VA <= rva < TEXT_VA+TEXT_SZ:
        return TEXT_RAW + (rva - TEXT_VA)
    return None

# 1) 收集诊断串
strings = {}  # rva -> label
for m in re.finditer(rb'\(GameReceiver::(\w+)\)[^\x00]{0,40}', data):
    rva = None
    off = m.start()
    # 反推 rva：先定位所在节
    # .rdata vaddr 0x3e000 raw 0x3d200; .text raw 0x400
    if 0x3d200 <= off < 0x3d200+0xee00:
        rva = 0x3e000 + (off - 0x3d200)
    elif 0x400 <= off < 0x400+0x3ce00:
        rva = 0x1000 + (off - 0x400)
    if rva is None:
        continue
    label = m.group(1).decode()
    strings.setdefault(rva, label)

# 2) .text 内扫 LEA rdx,[rip+X] 引用这些串
text = data[TEXT_RAW:TEXT_RAW+TEXT_SZ]
lea_sites = {}  # str_rva -> [(site)]
for i in range(len(text)-7):
    # 48 8d 1d / 48 8d 15 / 4c 8d 05 等 LEA rip
    if text[i] == 0x48 and text[i+1] == 0x8d and text[i+2] in (0x15, 0x1d, 0x05, 0x0d):
        rel = struct.unpack_from('<i', text, i+3)[0]
        site = TEXT_VA + i
        tgt = site + 7 + rel
        if tgt in strings:
            lea_sites.setdefault(tgt, []).append(site)

# 3) 函数边界（CC padding）归并
bounds = []
i = 0
while i < len(text):
    if text[i] == 0xCC and i+1 < len(text) and text[i+1] != 0xCC:
        bounds.append(TEXT_VA + i + 1)
    i += 1
import bisect
bounds.sort()

def enclosing(site):
    k = bisect.bisect_right(bounds, site) - 1
    return bounds[k] if k >= 0 else None

out = {}
for str_rva, sites in sorted(lea_sites.items()):
    for site in sites:
        fn = enclosing(site)
        out.setdefault(strings[str_rva], set()).add(fn)

for label in sorted(out, key=lambda s: min(out[s]) if out[s] else 0):
    fns = sorted(f for f in out[label] if f)
    print(f'{label:28s} {" ".join(hex(f) for f in fns)}')
