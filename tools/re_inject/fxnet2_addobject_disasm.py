# 反推注入帧数据（B 线）：fxnet2 ServerAddObject 全量反汇编驾驶舱。
# 目标：从程序事实定 0x0D/0x28 物化帧的权威布局 + 建对象闸。
# 用法: python fxnet2_addobject_disasm.py [rva_hex] [n_bytes]
import struct, sys
from capstone import Cs, CS_ARCH_X86, CS_MODE_64

DLL = r'G:/pojie/ai/9yin-server-go/artifacts/fxgame/fxnet2_x64.dll'
data = open(DLL, 'rb').read()

e = struct.unpack_from('<I', data, 0x3c)[0]
nsec = struct.unpack_from('<H', data, e+6)[0]
optsz = struct.unpack_from('<H', data, e+20)[0]
so = e+24+optsz
secs = []
for i in range(nsec):
    o = so+40*i
    name = data[o:o+8].rstrip(b'\x00').decode('latin1')
    vsize, vaddr, rsize, raddr = struct.unpack_from('<IIII', data, o+8)
    secs.append((vaddr, vsize, raddr, rsize, name))

def rva2off(rva):
    for vaddr, vsize, raddr, rsize, name in secs:
        if vaddr <= rva < vaddr+max(vsize, rsize):
            return raddr + (rva-vaddr)
    return None

# 收集 call 目标便于标注
md = Cs(CS_ARCH_X86, CS_MODE_64)
md.detail = True

KNOWN = {
    0x0E700: 'ServerAddObject',
    0x1bc90: 'record_parse(55-case)',
    0x1d6e0: 'object_table_gate?',
    0x178F0: 'ServerPropertyTable',
    0x0EF40: 'ServerObjectProperty',
    0x0E320: 'ServerEntryScene',
    0x17C60: 'ServerRecordTable',
    0x0BE60: 'value_pair_decode',
    0x0F020: 'name_hash_bkdr',
    0x01820: 'name_fmt',
}

def dis(rva, n_bytes=0x340, stop_at_ret=True):
    off = rva2off(rva)
    code = data[off:off+n_bytes]
    print(f';===== RVA {rva:#x} ({KNOWN.get(rva,"")}) file_off={off:#x} len={n_bytes:#x} =====')
    rets = 0
    for insn in md.disasm(code, rva):
        note = ''
        # call 目标标注
        if insn.mnemonic == 'call' and insn.operands[0].type == 2:  # IMM
            t = insn.operands[0].imm
            note = f'   ; -> {KNOWN.get(t, "")}'
        # 内存引用 [rip+X] 解析
        if 'rip' in insn.op_str:
            for op in insn.operands:
                if op.type == 3 and op.mem.base == 41:  # X86_REG_RIP=41 in capstone? use disp
                    pass
        print(f'{insn.address:#x}: {insn.mnemonic:9s} {insn.op_str}{note}')
        if insn.mnemonic == 'ret':
            rets += 1
            if stop_at_ret and rets >= 1:
                break
        if insn.mnemonic == 'int3':
            break

def calls_in(rva, n_bytes):
    """列函数内全部 call 目标（含重复），供调用图。"""
    off = rva2off(rva)
    code = data[off:off+n_bytes]
    seen = []
    for insn in md.disasm(code, rva):
        if insn.mnemonic == 'call' and insn.operands[0].type == 2:
            seen.append((insn.address, insn.operands[0].imm))
    return seen

def string_at(rva, maxn=64):
    off = rva2off(rva)
    if off is None: return None
    raw = data[off:off+maxn]
    nul = raw.find(b'\x00')
    if nul < 1: return None
    try:
        s = raw[:nul].decode('gbk')
        if all(31 < ord(c) < 0x3000 or c in '\\/. _-()%:' for c in s):
            return s
    except Exception:
        return None
    return None

if __name__ == '__main__':
    rva = int(sys.argv[1], 16) if len(sys.argv) > 1 else 0xE700
    n = int(sys.argv[2], 16) if len(sys.argv) > 2 else 0x340
    dis(rva, n, stop_at_ret=False)
