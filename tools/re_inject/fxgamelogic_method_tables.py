# fxgamelogic dump：Lua 方法表定位器——围绕指定串的 LEA 引用站，dump 同表的相邻方法名
import struct, sys, re
sys.path.insert(0, '.')
from fxgamelogic_dis import data, off, SECS
from capstone import Cs, CS_ARCH_X86, CS_MODE_64

BASE = 0x11000000
md = Cs(CS_ARCH_X86, CS_MODE_64)

def off2rva(fo):
    for vaddr, vsize, raddr, rsize in SECS:
        if raddr <= fo < raddr+rsize:
            return vaddr + (fo - raddr)
    return None

def rva2off(rva):
    return off(BASE + rva)

def cstr(rva, maxlen=48):
    o = rva2off(rva)
    if o is None:
        return None
    raw = data[o:o+maxlen]
    nul = raw.find(b'\x00')
    if nul < 1:
        return None
    try:
        s = raw[:nul].decode('ascii')
    except Exception:
        return None
    if all(32 <= ord(c) < 127 for c in s):
        return s
    return None

def lea_string_xrefs(str_rva):
    """扫 .text 中 lea reg,[rip+X] → str_rva 的站点。"""
    hits = []
    text = data[0x600:0x600+0x1cc5200]
    for m in re.finditer(rb'\x48\x8d[\x05\x0d\x15\x1d\x25\x2d\x35\x3d]', text):
        i = m.start()
        rel = struct.unpack_from('<i', text, i+3)[0]
        site_rva = 0x1000 + i + 7
        if site_rva + rel == str_rva:
            hits.append(site_rva)
    return hits

def nearby_strings(site_rva, span=0x180):
    """站点前后 span 字节内的全部 LEA 串引用（方法表=一段 lea rdx,[name] 序列）。"""
    o = rva2off(site_rva - span)
    code = data[o:o+2*span]
    base_rva = site_rva - span
    out = []
    for insn in md.disasm(code, base_rva):
        if insn.mnemonic == 'lea' and 'rip' in insn.op_str:
            m = re.search(r'rip ([+-]) (0x[0-9a-f]+)', insn.op_str)
            if m:
                delta = int(m.group(2), 16)
                if m.group(1) == '-':
                    delta = -delta
                tgt = insn.address + insn.size + delta
                s = cstr(tgt - BASE)
                if s and len(s) > 2:
                    out.append((insn.address, s))
    return out

if __name__ == '__main__':
    # AddObject 串：file 0x1ceeb18 -> rva
    str_rva = off2rva(0x1ceeb18)
    print(f'"AddObject" rva {str_rva:#x}')
    for site in lea_string_xrefs(str_rva):
        names = nearby_strings(site)
        label = ', '.join(n for _, n in names[:14])
        print(f'site {BASE+site:#x}: {label}')
