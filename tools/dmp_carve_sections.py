# -*- coding: utf-8 -*-
"""Carve fxgame.exe sections out of a MINIDUMP by VA.

内存里 PE 头被 VMP 抹掉，节表用磁盘文件的；VA 按模块运行时 base+节 RVA 映射。
用法: python dmp_carve_sections.py <dump路径> [输出目录] [磁盘exe路径]
"""
import os
import struct
import sys

DMP = sys.argv[1] if len(sys.argv) > 1 else r"F:\Game\Online\9yin\fxgame.DMP"
OUT = sys.argv[2] if len(sys.argv) > 2 else r"G:\pojie\ai\9yin-server-go\tmp_carve"
EXE = sys.argv[3] if len(sys.argv) > 3 else r"F:\Game\Online\9yin\Age of Wushu\bin64\fxgame.exe"
os.makedirs(OUT, exist_ok=True)

f = open(DMP, "rb")
hdr = f.read(32)
sig, ver, nstreams, dirrva = struct.unpack_from("<IIII", hdr, 0)
assert sig == 0x504D444D, "not MDMP"
print("dump=%s flags=0x%X" % (DMP, struct.unpack_from("<Q", hdr, 24)[0]))
f.seek(dirrva)
mem64 = modlist = None
for i in range(nstreams):
    st, ds, rva = struct.unpack("<III", f.read(12))
    if st == 9:
        mem64 = rva
    elif st == 4:
        modlist = rva

f.seek(mem64)
n, base_rva = struct.unpack("<QQ", f.read(16))
ranges = []
off = base_rva
for i in range(n):
    start, size = struct.unpack("<QQ", f.read(16))
    ranges.append((start, size, off))
    off += size
print("memory: %d ranges, %.2f GB" % (n, sum(s for _, s, _ in ranges) / 1024 ** 3))


def read_va(va, size):
    out = bytearray()
    pos, end = va, va + size
    while pos < end:
        hit = next(((s, sz, o) for s, sz, o in ranges if s <= pos < s + sz), None)
        if not hit:
            break
        s, sz, o = hit
        take = min(end - pos, s + sz - pos)
        f.seek(o + (pos - s))
        out += f.read(take)
        pos += take
    return bytes(out), len(out)


f.seek(modlist)
nmod, = struct.unpack("<I", f.read(4))
base = isize = None
for i in range(nmod):
    f.seek(modlist + 4 + 108 * i)
    b, isz, ck, td, nrva = struct.unpack("<QIIII", f.read(24))
    f.seek(nrva)
    ln, = struct.unpack("<I", f.read(4))
    nm = f.read(ln).decode("utf-16-le", "replace").rstrip("\0")
    if nm.lower().endswith("fxgame.exe"):
        base, isize = b, isz
print("fxgame base=0x%X SizeOfImage=0x%X" % (base, isize))

pe = open(EXE, "rb").read(0x120000)
e = struct.unpack_from("<I", pe, 0x3C)[0]
assert pe[e:e + 4] == b"PE\0\0"
mach, nsec, tds, psym, nsym, optsz, chars = struct.unpack_from("<HHIIIHH", pe, e + 4)
ep_rva = struct.unpack_from("<I", pe, e + 24 + 16)[0]
soff = e + 24 + optsz
print("disk PE: sections=%d EP_RVA=0x%X" % (nsec, ep_rva))

print("\n%-8s %-9s %-9s %-9s %-9s %-9s %s" % ("name", "VA", "VSZ", "RSZ(disk)", "captured", "density", "first16"))
code_total = 0
for i in range(nsec):
    s = pe[soff + 40 * i: soff + 40 * (i + 1)]
    name = s[:8].rstrip(b"\0").decode()
    vsz, va, rsz = struct.unpack_from("<III", s, 8)
    ch2 = struct.unpack_from("<I", s, 36)[0]
    data, got = read_va(base + va, vsz)
    if not data:
        print("%-8s 0x%07X 0x%07X 0x%07X (not captured)" % (name, va, vsz, rsz))
        continue
    nz = sum(1 for x in data if x)
    ep = " <== EP" if va <= ep_rva < va + max(vsz, 1) else ""
    print("%-8s 0x%07X 0x%07X 0x%07X 0x%07X %5.1f%%    %s%s" % (
        name, va, vsz, rsz, got, 100.0 * nz / max(got, 1),
        " ".join("%02X" % x for x in data[:12]), ep))
    open(os.path.join(OUT, "fxgame_%s.bin" % name), "wb").write(data)
    if ch2 & 0x20000000:
        code_total += got
print("\ncarved -> %s | code-attr sections: %.2f MB" % (OUT, code_total / 1048576.0))
