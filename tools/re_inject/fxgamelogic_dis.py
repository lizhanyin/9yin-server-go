# fxgamelogic.dll (Themida dump, ImageBase 0x11000000) 反汇编驾驶舱
import struct, sys
from capstone import Cs, CS_ARCH_X86, CS_MODE_64

DUMP = r'G:/pojie/ai/9yin-server-go/artifacts/fxgamelogic.dll'
data = open(DUMP, 'rb').read()
BASE = 0x11000000

# 节映射（dump 非平坦镜像）
_e = struct.unpack_from('<I', data, 0x3c)[0]
_nsec = struct.unpack_from('<H', data, _e+6)[0]
_optsz = struct.unpack_from('<H', data, _e+20)[0]
_so = _e+24+_optsz
SECS = []
for _i in range(_nsec):
    _o = _so+40*_i
    _vsize, _vaddr, _rsize, _raddr = struct.unpack_from('<IIII', data, _o+8)
    if _rsize:
        SECS.append((_vaddr, _vsize, _raddr, _rsize))

md = Cs(CS_ARCH_X86, CS_MODE_64)

def off(va):
    rva = va - BASE
    for vaddr, vsize, raddr, rsize in SECS:
        if vaddr <= rva < vaddr+max(vsize, rsize):
            fo = raddr + (rva - vaddr)
            if fo < len(data):
                return fo
    return None

def q(va):
    return struct.unpack_from('<Q', data, off(va))[0]

def dis(va, n_bytes=0x120, n_ins=70):
    code = data[off(va):off(va)+n_bytes]
    print(f';===== VA {va:#x} =====')
    cnt = 0
    for insn in md.disasm(code, va):
        print(f'{insn.address:#x}: {insn.mnemonic:9s} {insn.op_str}')
        cnt += 1
        if insn.mnemonic == 'ret' and cnt > 6:
            break
        if cnt >= n_ins:
            break

def vt_slot(vtable_va, slot):
    ptr = q(vtable_va + 8*slot)
    print(f'vtable {vtable_va:#x} slot {slot} (+{slot*8:#x}) -> {ptr:#x}')
    return ptr

if __name__ == '__main__':
    va = int(sys.argv[1], 16)
    n = int(sys.argv[2], 16) if len(sys.argv) > 2 else 0x120
    dis(va, n)
