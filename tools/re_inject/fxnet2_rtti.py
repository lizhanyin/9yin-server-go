# fxnet2 RTTI 链走查：类名 -> TypeDescriptor -> COL -> vtable -> 函数
import struct, sys
sys.path.insert(0, '.')
from fxnet2_addobject_disasm import rva2off

data = open(r'G:/pojie/ai/9yin-server-go/artifacts/fxgame/fxnet2_x64.dll','rb').read()
BASE = 0x10600000
# .rdata rva 0x3e000 raw 0x3d200 size 0xee00；.text 0x1000/0x400
def rva_off(rva):
    return rva2off(rva)

def find_typedesc(name):
    pat = ('.?AV' + name + '@@').encode()
    i = data.find(pat)
    while i != -1:
        fo = i - 0x10
        rva = off2rva(fo)
        if rva is not None:
            return rva
        i = data.find(pat, i+1)
    return None

SECS = []
_e = struct.unpack_from('<I', data, 0x3c)[0]
_nsec = struct.unpack_from('<H', data, _e+6)[0]
_optsz = struct.unpack_from('<H', data, _e+20)[0]
_so = _e+24+_optsz
for _i in range(_nsec):
    _o = _so+40*_i
    _vs, _va, _rs, _ra = struct.unpack_from('<IIII', data, _o+8)
    SECS.append((_vs, _va, _rs, _ra))

def off2rva(fo):
    for vs, va, rs, ra in SECS:
        if ra <= fo < ra+rs:
            return va + (fo - ra)
    return None

def rdata_qwords_equal(val):
    """全数据节扫指向 val(VA) 的 qword，返回命中处 RVA。"""
    hits = []
    for vs, va, rs, ra in SECS:
        if va in (0x1000,):  # 跳过 .text
            continue
        for fo in range(ra, min(ra+rs, len(data))-8, 4):
            if struct.unpack_from('<Q', data, fo)[0] == val:
                r = va + (fo - ra)
                hits.append(r)
    return hits

def walk(name):
    td_rva = find_typedesc(name)
    print(f'TypeDescriptor {name}: RVA {td_rva:#x} VA {BASE+td_rva:#x}')
    if not td_rva: return
    td_va = BASE + td_rva
    cols = rdata_qwords_equal(td_va)
    print(f'COL refs (pTypeDescriptor@COL+0xC): {[hex(c-0xC) for c in cols]}')
    for c in cols:
        col_va = BASE + c - 0xC
        vts = rdata_qwords_equal(col_va)
        for v in vts:
            print(f'  vtable@RVA {v:#x} (VA {BASE+v:#x}), vtable[0] = {struct.unpack_from("<Q", data, rva_off(v+8))[0]:#x}')

if __name__ == '__main__':
    for n in sys.argv[1:]:
        walk(n)
        print()
