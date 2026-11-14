# -*- coding: utf-8 -*-
"""Inspect D:\\fxgame.DMP: dump type flags, memory coverage, module list,
VMP section decrypt state at known fxgame RVAs, fxcli presence."""
import struct
import sys

P = r"D:\fxgame.DMP"
f = open(P, "rb")
hdr = f.read(32)
sig, ver, nstreams, dirrva, csum, tds, flags = struct.unpack("<IIIIIIQ", hdr[:32])
print("sig=%s streams=%d dirRva=0x%X time=%d" % (sig, nstreams, dirrva, tds))
print("Flags=0x%016X" % flags)
FBITS = {0x1: "WithDataSegs", 0x2: "WithFullMemory", 0x4: "WithHandleData", 0x8: "FilterMemory",
         0x10: "ScanMemory", 0x20: "WithUnloadedModules", 0x40: "WithIndirectlyReferencedMemory",
         0x80: "FilterModulePaths", 0x100: "StopOnException", 0x200: "WithFullAuxiliaryState",
         0x400: "WithPrivateWriteCopyMemory", 0x800: "IgnoreInaccessibleMemory",
         0x1000: "WithTokenInformation"}
print("  " + " | ".join(n for b, n in FBITS.items() if flags & b) or "(minimal)")

f.seek(dirrva)
dirs = []
for i in range(nstreams):
    st, ds, rva = struct.unpack("<III", f.read(12))
    dirs.append((st, ds, rva))
STN = {3: "ThreadList", 4: "ModuleList", 5: "MemoryList", 6: "Exception", 7: "SystemInfo",
       9: "Memory64List", 10: "CommentA", 12: "FunctionTable", 13: "MemoryInfoList",
       15: "ThreadInfoList", 16: "HandleData", 18: "UnloadedModuleList", 22: "SystemMemoryInfo"}
mem64 = memlist = modlist = None
for st, ds, rva in dirs:
    print("stream type=%d (%s) size=0x%X rva=0x%X" % (st, STN.get(st, "?"), ds, rva))
    if st == 9:
        mem64 = rva
    elif st == 5:
        memlist = (ds, rva)
    elif st == 4:
        modlist = rva

# --- Memory64 coverage ---------------------------------------------------
ranges = []
if mem64:
    f.seek(mem64)
    n, base_rva = struct.unpack("<QQ", f.read(16))
    print("\nMemory64List: %d ranges, data starts 0x%X" % (n, base_rva))
    off = base_rva
    for i in range(n):
        start, size = struct.unpack("<QQ", f.read(16))
        ranges.append((start, size, off))
        off += size
    total = sum(s for _, s, _ in ranges)
    print("total memory captured: %.2f GB in %d ranges" % (total / 1024 ** 3, n))


def va2off(va):
    for start, size, off in ranges:
        if start <= va < start + size:
            return off + (va - start)
    return None


def sample(va, n=4096, label=""):
    o = va2off(va)
    if o is None:
        return None
    f.seek(o)
    d = f.read(n)
    nz = sum(1 for b in d if b)
    return d, nz


# --- Module list ---------------------------------------------------------
f.seek(modlist)
nmod, = struct.unpack("<I", f.read(4))
print("\nmodules in dump: %d" % nmod)
mods = {}
modtbl = modlist + 4
for i in range(nmod):
    f.seek(modtbl + 108 * i)
    base, size, ck, tds, name_rva = struct.unpack("<QIIII", f.read(24))
    f.seek(name_rva)
    ln, = struct.unpack("<I", f.read(4))
    name = f.read(ln).decode("utf-16-le", "replace").rstrip("\0")
    mods[name.lower()] = (base, size)
    if any(k in name.lower() for k in ("fxgame", "fxcli", "fxprotect", "fxgamelogic", ".sn", "spdc", "fxnet", "fxcore", "fxrender")):
        print("  %-40s base=0x%012X size=0x%X" % (name, base, size))

# --- fxgame decrypt checks ----------------------------------------------
fg = mods.get("fxgame.exe")
if fg and ranges:
    base, size = fg
    print("\nfxgame.exe in dump: base=0x%X size=0x%X" % (base, size))
    # 1) PE header region (expect wiped tail)
    r = sample(base + 0x80, 64, "PE sig")
    print("PE sig @+0x80      :", repr(r[0][:8]) if r else "(not captured)")
    r2 = sample(base + 0x100, 256, "header tail")
    print("header tail @+0x100:", "captured, nonzero=%d/256" % r2[1] if r2 else "(not captured)")
    # 2) entry point region .vmp1 EP_RVA 0x977E30
    for rva, tag in ((0x977E30, "EP .vmp1"), (0x1000, "RVA 0x1000 (old .text?)"),
                     (0x900000, "0x900000 (vmp1 zone)"), (0x960000, "0x960000 (vmp1 zone)")):
        r = sample(base + rva, 4096)
        if r:
            d, nz = r
            print("%-26s: nonzero=%4d/4096  first=%s" % (tag, nz, " ".join("%02X" % b for b in d[:12])))
        else:
            print("%-26s: (VA not captured)" % tag)

# --- fxcli key table check ----------------------------------------------
fc = mods.get("fxcli.dll")
if fc and ranges:
    base, size = fc
    print("\nfxcli.dll in dump: base=0x%X size=0x%X" % (base, size))
    r = sample(base + 0xC9700, 256, "xor key table 0xC9700")
    if r:
        d, nz = r
        print("XOR key table @+0xC9700: nonzero=%d/256 first=%s" % (nz, " ".join("%02X" % b for b in d[:16])))
    r = sample(base + 0xF8B3C, 128, "key table 0xF8B3C")
    if r:
        d, nz = r
        print("key table   @+0xF8B3C: nonzero=%d/128 first=%s" % (nz, " ".join("%02X" % b for b in d[:16])))
