# -*- coding: utf-8 -*-
"""Full-dump anchor sweep: stream all captured memory, find anchor strings,
map file offset -> VA -> owning module (by image span)."""
import struct
import sys

DMP = sys.argv[1] if len(sys.argv) > 1 else r"F:\Game\Online\9yin\fxgame.DMP"
KEYS = [b".?AVGameMessageHandler@@", b".?AVIGameMsgHandler@@", b"GameReceiver",
        b".?AVGameReceiver@@", b"NewPacket", b"GameMsgHandler", b"SERVER_",
        b"OnObjectUpdate", b"PROPERTY_TABLE", b"ServerAddObject", b"GameMessage",
        b".?AVSceneObjRecord@@", b"RECORD_TABLE", b"ENTRY_SCENE"]

f = open(DMP, "rb")
hdr = f.read(32)
nstreams, dirrva = struct.unpack_from("<II", hdr, 8)
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

f.seek(modlist)
nmod, = struct.unpack("<I", f.read(4))
mods = []
for i in range(nmod):
    f.seek(modlist + 4 + 108 * i)
    b, isz, ck, td, nrva = struct.unpack("<QIIII", f.read(24))
    f.seek(nrva)
    ln, = struct.unpack("<I", f.read(4))
    nm = f.read(ln).decode("utf-16-le", "replace").rstrip("\0")
    mods.append((b, isz, nm.split("\\")[-1]))
mods.sort()


def owner(va):
    for b, isz, nm in mods:
        if b <= va < b + isz:
            return "%s+0x%X" % (nm, va - b)
    return "heap/other @0x%X" % va


def off2va(o):
    for start, size, fo in ranges:
        if fo <= o < fo + size:
            return start + (o - fo)
    return None


# stream file from base_rva to end, chunked with overlap
import io
CH = 1 << 24
OV = max(len(k) for k in KEYS)
f.seek(base_rva)
pos = base_rva
prev_tail = b""
hits = {k: [] for k in KEYS}
while True:
    buf = f.read(CH)
    if not buf:
        break
    data = prev_tail + buf
    base_off = pos - len(prev_tail)
    for k in KEYS:
        start = 0
        while True:
            i = data.find(k, start)
            if i < 0:
                break
            va = off2va(base_off + i)
            if va:
                hits[k].append(va)
            start = i + 1
    prev_tail = data[-OV:]
    pos += len(buf)

for k in KEYS:
    v = hits[k]
    if v:
        owners = {}
        for va in v:
            o = owner(va)
            owners.setdefault(o, []).append(va)
        print("%-26s total=%d" % (k.decode(), len(v)))
        for o, vas in sorted(owners.items(), key=lambda x: -len(x[1]))[:8]:
            print("   %-34s x%d  %s" % (o, len(vas), " ".join("0x%X" % x for x in vas[:4])))
    else:
        print("%-26s (0)" % k.decode())
