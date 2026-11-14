# fxnet2 .text 全量 E8/E9 交叉引用扫描（壳内重排后文档地址系失准，用 xref 重建调用图）
import struct, sys
from capstone import Cs, CS_ARCH_X86, CS_MODE_64

DLL = r'G:/pojie/ai/9yin-server-go/artifacts/fxgame/fxnet2_x64.dll'
data = open(DLL, 'rb').read()
TEXT_VA, TEXT_SZ, TEXT_RAW = 0x1000, 0x3ccea, 0x400

def rva2off(rva):
    return TEXT_RAW + (rva - TEXT_VA)

md = Cs(CS_ARCH_X86, CS_MODE_64)

def build_xrefs():
    """线性反汇编全 .text（对齐敏感，用启发：同时扫 E8/E9 rel32 原始字节）。"""
    xrefs = {}  # target -> [(site, kind)]
    blob = data[TEXT_RAW:TEXT_RAW+TEXT_SZ]
    for i in range(len(blob)-5):
        op = blob[i]
        if op in (0xE8, 0xE9):
            rel = struct.unpack_from('<i', blob, i+1)[0]
            site = TEXT_VA + i
            tgt = site + 5 + rel
            if TEXT_VA <= tgt < TEXT_VA+TEXT_SZ:
                xrefs.setdefault(tgt, []).append((site, 'call' if op == 0xE8 else 'jmp'))
    return xrefs

if __name__ == '__main__':
    targets = [int(a, 16) for a in sys.argv[1:]]
    xr = build_xrefs()
    for t in targets:
        print(f'target {t:#x}:')
        for site, kind in xr.get(t, []):
            print(f'  {kind} @ {site:#x}')
        if not xr.get(t):
            print('  (no direct xrefs)')
