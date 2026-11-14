# -*- coding: utf-8 -*-
"""fxnet2 GameReceiver 静态定位：
1) 诊断串/名字串 RVA -> .text 中 rip-LEA 引用点 -> 所在函数(按 CC 边界)
2) E8 直调扫描 -> 调用多个 Server* 函数的公共调用者 = op 分发器候选
3) 分发器附近 dump 找 op 常量
"""
import re
import struct

P = r"F:\Game\Online\9yin\Age of Wushu\bin64\fxnet2.dll"
d = open(P, "rb").read()
e = struct.unpack_from("<I", d, 0x3C)[0]
nsec = struct.unpack_from("<H", d, e + 6)[0]
optsz = struct.unpack_from("<H", d, e + 20)[0]
secs = []
for i in range(nsec):
    s = d[e + 24 + optsz + 40 * i:e + 24 + optsz + 40 * (i + 1)]
    vsz, va, rsz, praw = struct.unpack_from("<IIII", s, 8)
    secs.append((s[:8].rstrip(b"\0").decode("latin1"), va, vsz, rsz, praw))

TEXT_RVA, TEXT_OFF, TEXT_SZ = 0x1000, 0x400, 0x3F200
td = d[TEXT_OFF:TEXT_OFF + TEXT_SZ]


def rva2off(rva):
    for nm, va, vsz, rsz, praw in secs:
        if va <= rva < va + max(vsz, rsz) and rva - va < rsz:
            return praw + (rva - va)


def off2rva(off):
    for nm, va, vsz, rsz, praw in secs:
        if praw <= off < praw + rsz:
            return va + (off - praw)


# ---- 1) 目标字符串 RVA -----------------------------------------------
targets = {}  # rva -> label
for m in re.finditer(rb"[\x20-\x7e]{4,}", d):
    s = m.group().decode()
    if s.startswith("(GameReceiver::Server") or s.startswith("Server") and len(s) < 60:
        rva = off2rva(m.start())
        if rva:
            targets[rva] = s

# ---- 2) rip-LEA 引用 ---------------------------------------------------
# 48 8D /r  或 4C 8D /r，modrm.rm=101 -> rel32
lea_refs = {}  # str_rva -> [site_rva...]
i = 0
while i < TEXT_SZ - 7:
    op0, op1 = td[i], td[i + 1]
    if op0 in (0x48, 0x4C) and op1 == 0x8D:
        modrm = td[i + 2]
        if (modrm & 0xC7) == 0x05:
            rel = struct.unpack_from("<i", td, i + 3)[0]
            tgt = TEXT_RVA + i + 7 + rel
            if tgt in targets:
                lea_refs.setdefault(tgt, []).append(TEXT_RVA + i)
            i += 7
            continue
    i += 1

# ---- 3) LEA 点 -> 所在函数(向前找 CC CC 边界) --------------------------
def func_start(rva):
    off = rva - TEXT_RVA
    j = td.rfind(b"\xCC\xCC", 0, off)
    if j < 0:
        j = -1
    k = j + 2
    while k < off and td[k] == 0xCC:
        k += 1
    return TEXT_RVA + k


funcs = {}  # func_rva -> {label: [strs]}
for str_rva, sites in lea_refs.items():
    for s in sites:
        f = func_start(s)
        funcs.setdefault(f, {"strs": [], "sites": []})
        funcs[f]["strs"].append(targets[str_rva])
        funcs[f]["sites"].append(s)

print("Server* 函数定位: %d 个（含诊断串 LEA）" % len(funcs))
srv_starts = []
for f in sorted(funcs):
    labs = sorted(set(x.split(")")[0] + ")" if ")" in x else x for x in funcs[f]["strs"]))
    main = [x for x in labs if "::Server" in x or x.startswith("Server")]
    if not main:
        continue
    srv_starts.append(f)
    print("  fn 0x%06X  %-46s refs=%d  firstlea=0x%X" % (f, main[0][:46], len(funcs[f]["sites"]), funcs[f]["sites"][0]))

# ---- 4) E8 直调 -> 公共调用者 ------------------------------------------
callers = {}  # caller_func -> set(called)
srv_set = set(srv_starts)
for m in re.finditer(rb"\xE8", td):
    off = m.start()
    rel = struct.unpack_from("<i", td, off + 1)[0]
    tgt = TEXT_RVA + off + 5 + rel
    if tgt in srv_set:
        c = func_start(TEXT_RVA + off)
        callers.setdefault(c, set()).add(tgt)

print("\n公共调用者（按调到的 Server* 数排序）:")
for c, called in sorted(callers.items(), key=lambda x: -len(x[1]))[:12]:
    print("  caller 0x%06X calls %d Server* fns: %s" % (
        c, len(called), " ".join("0x%X" % t for t in sorted(called)[:12])))
