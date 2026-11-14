# 围绕锚点站原始字节扫 LEA→串，识别 Lua 方法注册簇的属主类
import struct, sys
sys.path.insert(0, '.')
from fxgamelogic_dis import data, off, SECS

BASE = 0x11000000

def off2rva(fo):
    for vaddr, vsize, raddr, rsize in SECS:
        if raddr <= fo < raddr+rsize:
            return vaddr + (fo - raddr)
    return None

def cstr_at_rva(rva):
    for vaddr, vsize, raddr, rsize in SECS:
        # 数据节（非 .text/.pdata）
        if vaddr in (0x1000, 0x241e000) or 'pdata' in str(vaddr):
            continue
        if vaddr <= rva < vaddr + vsize:
            fo = raddr + (rva - vaddr)
            raw = data[fo:fo+44]
            nul = raw.find(b'\x00')
            if nul < 3:
                return None
            try:
                s = raw[:nul].decode('ascii')
            except Exception:
                return None
            if all(32 <= ord(c) < 127 for c in s):
                return s
    return None

def scan_site(site_rva, span=0x200):
    """site_rva 附近 ±span 原始扫 48 8d / 4c 8d LEA，目标解析成可读串。"""
    out = []
    fo = off(BASE + site_rva)
    lo = max(fo - span, 0x600)
    for i in range(lo, fo + span):
        if data[i] in (0x48, 0x4C) and data[i+1] == 0x8D and (data[i+2] & 0xC7) == 0x05:
            rel = struct.unpack_from('<i', data, i+3)[0]
            site = off2rva(i) + 7
            tgt = site + rel
            s = cstr_at_rva(tgt)
            if s:
                out.append((BASE + site, s))
    return out

if __name__ == '__main__':
    sites = [0x2d29aa, 0x2d5183, 0x2d54c1, 0x2d5b4f, 0x4cdd09, 0x52d2d1, 0x52d48a,
             0x52dbca, 0x53a8b7, 0x7269cd, 0x8eeefb, 0x8f0b95, 0x8f24fb, 0xb5e3a4]
    for s in sites:
        names = scan_site(s)
        uniq = list(dict.fromkeys(n for _, n in names))
        print(f'site {BASE+s:#x}: {", ".join(uniq[:18])}')
