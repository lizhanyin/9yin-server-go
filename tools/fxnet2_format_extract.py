# -*- coding: utf-8 -*-
"""fxnet2 帧格式提取（最终版）：指针传播 + 读采集，匹配 op_str（不含助记符）。"""
from capstone import *
import re

d = open(r"F:\Game\Online\9yin\Age of Wushu\bin64\fxnet2.dll", "rb").read()
IMGBASE = 0x10600000
md = Cs(CS_ARCH_X86, CS_MODE_64)
HELPERS = {0x1820: "fmt名", 0xBE60: "★值对", 0xADA0: "双名查", 0x1730: "LZ解压"}

OPS = [(0x01,0x16480,"SetVerify"),(0x02,0x167E0,"SetEncode"),(0x03,0x16910,"Error"),
(0x04,0x16A80,"LoginSucceed"),(0x05,0x175F0,"WorldInfo"),(0x07,0x195D0,"Queue"),
(0x0B,0xE320,"EntryScene"),(0x0D,0xE700,"AddObject"),(0x0E,0xEA40,"RemoveObject"),
(0x0F,0xED40,"SceneProperty"),(0x11,0x10520,"RecordAddRow"),(0x12,0x11230,"RecordDelRow"),
(0x13,0x122C0,"RecordGrid"),(0x14,0x12F10,"RecordClear"),(0x15,0xF450,"CreateView"),
(0x16,0xF6D0,"DeleteView"),(0x17,0xF8C0,"ViewProperty"),(0x18,0xFBC0,"ViewAdd"),
(0x19,0xFF10,"ViewRemove"),(0x1A,0x18110,"Speech"),(0x1B,0x18300,"SystemInfo"),
(0x1C,0x184A0,"Menu"),(0x1D,0x18970,"ClearMenu"),(0x1E,0x18B40,"Custom"),
(0x1F,0x13960,"Location"),(0x20,0x13C80,"Moving"),(0x21,0x13FA0,"AllDest"),
(0x22,0x19280,"Warning"),(0x23,0x19420,"FromGmcc"),(0x24,0x14430,"LinkTo"),
(0x25,0x14B90,"Unlink"),(0x26,0x147E0,"LinkMove"),(0x2B,0x101F0,"ViewChange"),
(0x2D,0x14E20,"AllProp"),(0x2F,0x15390,"AddMoreObject"),(0x31,0x15930,"RemoveMoreObject"),
(0x32,0x15E60,"ChargeValidstring"),(0x37,0x16230,"SetQR")]

RE_MOVZX = re.compile(r"^\w+, (byte|word) ptr \[(\w+)(?: \+ (0x[0-9a-f]+|\d+))?\]$")
RE_DWORD = re.compile(r"^\w+, dword ptr \[(\w+)(?: \+ (0x[0-9a-f]+|\d+))?\]$")
RE_QWORD = re.compile(r"^\w+, qword ptr \[(\w+)(?: \+ (0x[0-9a-f]+|\d+))?\]$")
RE_LEA = re.compile(r"^(\w+), \[(\w+)(?: \+ (0x[0-9a-f]+|\d+))?\]$")
RE_ADD = re.compile(r"^(\w+), (0x[0-9a-f]+|\d+)$")
RE_CALL = re.compile(r"^0x([0-9a-f]+)$")

for op, frva, name in OPS:
    off = 0x400 + frva - 0x1000
    insns = list(md.disasm(d[off:off+0x400], IMGBASE+frva))
    ptr = {"rdx": 0}
    reads = []
    for i, ins in enumerate(insns[:150]):
        m, s = ins.mnemonic, ins.op_str
        if m == "mov":
            parts = [x.strip() for x in s.split(",")]
            if len(parts) == 2 and parts[1] == "rdx":
                ptr[parts[0]] = 0
            elif len(parts) == 2 and parts[1] in ptr and parts[0] != parts[1]:
                ptr[parts[0]] = ptr[parts[1]]
            mm = RE_DWORD.match(s)
            if mm and mm.group(1) in ptr:
                reads.append((ptr[mm.group(1)] + int(mm.group(2) or "0", 0), "u32"))
            mm = RE_QWORD.match(s)
            if mm and mm.group(1) in ptr:
                reads.append((ptr[mm.group(1)] + int(mm.group(2) or "0", 0), "u64"))
        elif m == "movzx":
            mm = RE_MOVZX.match(s)
            if mm and mm.group(2) in ptr:
                reads.append((ptr[mm.group(2)] + int(mm.group(3) or "0", 0),
                              {"byte": "u8", "word": "u16"}[mm.group(1)]))
        elif m == "lea":
            mm = RE_LEA.match(s)
            if mm and mm.group(2) in ptr:
                ptr[mm.group(1)] = ptr[mm.group(2)] + (int(mm.group(3), 0) if mm.lastindex >= 3 else 0)
        elif m == "add":
            mm = RE_ADD.match(s)
            if mm and mm.group(1) in ptr:
                ptr[mm.group(1)] += int(mm.group(2), 0)
                reads.append((None, "cursor->@0x%X" % ptr[mm.group(1)]))
        if m == "call":
            mm = RE_CALL.match(s)
            if mm:
                t = int(mm.group(1), 16) - IMGBASE
                if t in HELPERS:
                    reads.append((None, HELPERS[t]))
        if s == "rdi, rcx":
            reads.append((None, "NUL串"))
    seen, out = set(), []
    for o, k in reads:
        if (o, k) in seen:
            continue
        seen.add((o, k))
        out.append(("@0x%X:%s" % (o, k)) if o is not None else k)
    print("0x%02X %-17s %s" % (op, name, " ".join(out[:15])))
