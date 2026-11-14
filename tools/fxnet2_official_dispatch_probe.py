# -*- coding: utf-8 -*-
"""官服 fxnet2.dll GameReceiver 静态定位（fork 版 fxnet2_serverxrefs.py 的参数化移植）：
诊断串/名字串 RVA -> .text rip-LEA 引用 -> 所在函数 -> E8 公共调用者 = op 分发器候选。
用法: python -I fxnet2_official_dispatch_probe.py <fxnet2.dll 路径>
"""
import re
import struct
import sys

P = sys.argv[1] if len(sys.argv) > 1 else r"F:\Game\Online\9yin\9yin\bin64\fxnet2.dll"
d = open(P, "rb").read()
e = struct.unpack_from("<I", d, 0x3C)[0]
nsec = struct.unpack_from("<H", d, e + 6)[0]
optsz = struct.unpack_from("<H", d, e + 20)[0]
secs = []
for i in range(nsec):
    s = d[e + 24 + optsz + 40 * i:e + 24 + optsz + 40 * (i + 1)]
    vsz, va, rsz, praw = struct.unpack_from("<IIII", s, 8)
    name = s[:8].rstrip(b"\0").decode("latin1")
    secs.append((name, va, vsz, rsz, praw))
    print("sec %-8s va=%#x vsz=%#x rsz=%#x raw=%#x" % (name, va, vsz, rsz, praw))


def rva2off(rva):
    for nm, va, vsz, rsz, praw in secs:
        if va <= rva < va + max(vsz, rsz) and rva - va < rsz:
            return praw + (rva - va)


def off2rva(off):
    for nm, va, vsz, rsz, praw in secs:
        if praw <= off < praw + rsz:
            return va + (off - praw)


text = next((s for s in secs if s[0] == ".text"), None)
TEXT_RVA, TEXT_OFF, TEXT_SZ = text[1], text[4], text[3]
td = d[TEXT_OFF:TEXT_OFF + TEXT_SZ]
print(".text rva=%#x off=%#x size=%#x" % (TEXT_RVA, TEXT_OFF, TEXT_SZ))

targets = {}
for m in re.finditer(rb"[\x20-\x7e]{4,}", d):
    s = m.group().decode()
    if s.startswith("(GameReceiver::Server") or (s.startswith("Server") and len(s) < 60):
        rva = off2rva(m.start())
        if rva:
            targets[rva] = s

lea_refs = {}
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


def func_start(rva):
    off = rva - TEXT_RVA
    j = td.rfind(b"\xCC\xCC", 0, off)
    if j < 0:
        j = -1
    k = j + 2
    while k < off and td[k] == 0xCC:
        k += 1
    return TEXT_RVA + k


funcs = {}
for str_rva, sites in lea_refs.items():
    for s in sites:
        f = func_start(s)
        funcs.setdefault(f, {"strs": [], "sites": []})
        funcs[f]["strs"].append(targets[str_rva])
        funcs[f]["sites"].append(s)

print("Server* 函数定位: %d 个" % len(funcs))
srv_starts = {}
for f in sorted(funcs):
    labs = sorted(set(x.split(")")[0] + ")" if ")" in x else x for x in funcs[f]["strs"]))
    main = [x for x in labs if "::Server" in x or x.startswith("Server")]
    if not main:
        continue
    srv_starts[f] = main[0]
    print("  fn 0x%06X  %-52s refs=%d" % (f, main[0][:52], len(funcs[f]["sites"])))

callers = {}
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
