# -*- coding: utf-8 -*-
"""Rebuild an analyzable fxgame.exe from disk skeleton + carved runtime memory.

骨架=磁盘 PE(头/节表/目录全有效)；节 raw=运行时 dump 抠取字节(VMP 已解密)。
输出: tmp_carve/fxgame_rebuilt.exe + tmp_carve/fxgame_functions.csv(.pdata 函数边界)
"""
import os
import struct

EXE = r"F:\Game\Online\9yin\Age of Wushu\bin64\fxgame.exe"
CARVE = r"G:\pojie\ai\9yin-server-go\tmp_carve"
OUT = os.path.join(CARVE, "fxgame_rebuilt.exe")

disk = open(EXE, "rb").read()
e = struct.unpack_from("<I", disk, 0x3C)[0]
mach, nsec, tds, psym, nsym, optsz, chars = struct.unpack_from("<HHIIIHH", disk, e + 4)
magic = struct.unpack_from("<H", disk, e + 24)[0]
filealign, secalign = struct.unpack_from("<II", disk, e + 24 + 36)
ep_rva = struct.unpack_from("<I", disk, e + 24 + 16)[0]
soff = e + 24 + optsz
print("skeleton: nsec=%d optsz=0x%X filealign=0x%X secalign=0x%X EP=0x%X" % (nsec, optsz, filealign, secalign, ep_rva))

# ---- collect sections: (name, VSZ, VA, flags, runtime bytes) ----------
secs = []
for i in range(nsec):
    s = disk[soff + 40 * i: soff + 40 * (i + 1)]
    name = s[:8].rstrip(b"\0").decode()
    vsz, va, rsz, praw = struct.unpack_from("<IIII", s, 8)
    ch = struct.unpack_from("<I", s, 36)[0]
    p = os.path.join(CARVE, "fxgame_%s.bin" % name)
    data = open(p, "rb").read() if os.path.exists(p) else None
    if data is None:
        print("!! no carved bin for %s (skipped, keep disk raw)" % name)
        secs.append((name, vsz, va, ch, disk[praw:praw + rsz] if rsz else b""))
        continue
    assert len(data) == vsz or len(data) >= vsz, (name, len(data), vsz)
    secs.append((name, vsz, va, ch, data[:vsz]))

# ---- layout new file ---------------------------------------------------
hdr_size = soff + 40 * nsec
hdr_pad = (hdr_size + filealign - 1) // filealign * filealign
out = bytearray(disk[:hdr_size]) + b"\0" * (hdr_pad - hdr_size)

new_table = bytearray()
cursor = hdr_pad
for name, vsz, va, ch, data in secs:
    rsz = (len(data) + filealign - 1) // filealign * filealign
    pad = rsz - len(data)
    out += data + b"\0" * pad
    new_table += name.encode().ljust(8, b"\0")
    new_table += struct.pack("<IIII", vsz, va, rsz, cursor) + b"\0" * 12 + struct.pack("<I", ch)
    print("  %-8s VA=0x%07X raw 0x%X @ file 0x%X" % (name, va, rsz, cursor))
    cursor += rsz

# patch section table
out[soff:soff + 40 * nsec] = new_table
# zero checksum
struct.pack_into("<I", out, e + 24 + 64, 0)

open(OUT, "wb").write(bytes(out))
print("\nwrote %s (%.2f MB)" % (OUT, len(out) / 1048576.0))

# ---- validate: reparse -------------------------------------------------
v = open(OUT, "rb").read()
ev = struct.unpack_from("<I", v, 0x3C)[0]
assert v[ev:ev + 4] == b"PE\0\0"
nv = struct.unpack_from("<H", v, ev + 6)[0]
ov = struct.unpack_from("<H", v, ev + 20)[0]
sv = ev + 24 + ov
ok = True
ep_bytes = b""
for i in range(nv):
    s = v[sv + 40 * i: sv + 40 * (i + 1)]
    name = s[:8].rstrip(b"\0").decode("latin1")
    vsz, va, rsz, praw = struct.unpack_from("<IIII", s, 8)
    orig = dict((n, d) for n, _, _, _, d in secs)[name]
    got = v[praw:praw + len(orig)]
    match = got == orig
    ok &= match
    print("validate %-8s %s" % (name, "OK" if match else "MISMATCH"))
    if va <= ep_rva < va + max(vsz, 1):
        ep_bytes = v[praw + (ep_rva - va):praw + (ep_rva - va) + 16]
print("EP bytes @RVA 0x%X: %s" % (ep_rva, " ".join("%02X" % b for b in ep_bytes)))
print("ALL SECTIONS VERIFIED" if ok else "VERIFICATION FAILED")

# ---- function list from .pdata ----------------------------------------
pd = dict((n, d) for n, _, _, _, d in secs)[".pdata"]
rows = []
for off in range(0, len(pd) - 11, 12):
    b, en, uw = struct.unpack_from("<III", pd, off)
    if b == 0:
        break
    rows.append((b, en, en - b))
csv = os.path.join(CARVE, "fxgame_functions.csv")
with open(csv, "w") as w:
    w.write("begin_rva,end_rva,size,begin_va\n")
    for b, en, sz in rows:
        w.write("0x%X,0x%X,%d,0x%X\n" % (b, en, sz, 0x140000000 + b))
print("functions from .pdata: %d -> %s" % (len(rows), csv))
import collections
zones = collections.Counter()
for b, en, sz in rows:
    if 0x8E0000 <= b < 0x8E0000 + 0x43D232:
        zones[".vmp0"] += 1
    elif 0x4CC000 <= b < 0x4CC000 + 0x64E81C:
        zones[".vmp1"] += 1
    elif b < 0x6D000:
        zones[".text"] += 1
    else:
        zones["other"] += 1
print("function distribution:", dict(zones))
