# npc.pcapng 原生 0x46 NPC 帧 → 回放包（[u32 len][frame]×N）+ 程序推导解析器逐帧验证
# 验证规则 = fxnet2 反汇编定案（docs/official/注入帧数据-AddObject全链程序反推.md）：
#   包装 [0x28][0x46][len-1 u32@2][outlen u32@6][体@10]，outlen 必须 == len-10
#   体 = [obj@0][own@4][posA@8][posB@0x18][scale@0x28][u32 标志@0x38][count u16@0x3C][值对@0x3E]
#   值对 = idx u16 + 0x09 官方 2433 表线型宽；idx 越表/走查失步 = 帧坏（上轮判别翻车根源）
import sys, struct
sys.path.insert(0, r'G:\pojie\ai\9yin-server-go-2433\tools\pcap_analysis')
sys.path.insert(0, r'G:\pojie\ai\9yin-server-go')
from pcaplib import build_streams, stream_frames

PCAP = r'F:/Game/Online/9yin/9yin-server-go/artifacts/npc.pcapng'
OUT = r'G:/pojie/ai/9yin-server-go/tmp/npc_native46_replay.bin'

# 官方 2433 表线型（idx → TypeCode）：从 Go 侧导出的表文件
TABLE = r'G:\pojie\ai\9yin-server-go\internal\clientdata\official_property_table_2433.txt'

def load_types(path):
    types = {}
    for line in open(path, encoding='utf-8-sig', errors='replace'):
        line = line.strip()
        if not line or line.startswith('#'):
            continue
        parts = line.split('\t') if '\t' in line else line.split(',')
        if len(parts) >= 3:
            try:
                idx = int(parts[0])
            except ValueError:
                continue
            # 列型码：1 byte 2 word 3 i32 4 i64 5 f32 7 串 8 宽串 9 obj
            tc = parts[2].strip()
            types[idx] = tc
    return types

WIDTH = {'1': 1, '2': 2, '3': 4, '4': 8, '5': 4, '7': None, '8': None, '9': 8}

def validate_frame(frame, types):
    errs = []
    if frame[0] != 0x28 or frame[1] != 0x46:
        return ['not 0x28/0x46']
    outlen = struct.unpack_from('<I', frame, 6)[0]
    if outlen != len(frame) - 10:
        errs.append(f'outlen {outlen} != {len(frame)-10}')
    body = frame[10:]
    if len(body) < 0x3E:
        errs.append(f'body {len(body)} < 0x3E')
        return errs
    count = struct.unpack_from('<H', body, 0x3C)[0]
    off = 0x3E
    fields = []
    for i in range(count):
        if off + 2 > len(body):
            errs.append(f'field#{i} idx 越体')
            break
        idx = struct.unpack_from('<H', body, off)[0]
        off += 2
        tc = types.get(idx)
        if tc is None:
            errs.append(f'field#{i} idx={idx} 不在 2433 表')
            break
        w = WIDTH.get(tc, None)
        if tc in ('7', '8'):
            if off + 4 > len(body):
                errs.append(f'field#{i} 串长越体')
                break
            n = struct.unpack_from('<I', body, off)[0]
            off += 4 + n
        else:
            off += w or 0
        fields.append((idx, tc))
    if off != len(body):
        errs.append(f'走查终点 {off} != 体长 {len(body)}（失步）')
    return errs, fields

if __name__ == '__main__':
    types = load_types(TABLE)
    print(f'2433 表 {len(types)} 行线型')
    streams = build_streams(PCAP)
    pack = bytearray()
    n_ok = 0
    for st in streams:
        if not st.s2c:
            continue
        for idx, (ts, p) in enumerate(stream_frames(st.s2c)):
            if p and p[0] == 0x28 and len(p) >= 10 and p[1] == 0x46:
                obj, own = struct.unpack_from('<II', p, 10)
                res = validate_frame(p, types)
                errs, fields = res if isinstance(res, tuple) else (res, [])
                status = 'OK ' if not errs else 'BAD'
                print(f'#{idx} {status} obj={obj:#x} own={own:#x} len={len(p)} 字段数={len(fields)}')
                if errs:
                    for e in errs:
                        print('    ', e)
                else:
                    pack += struct.pack('<I', len(p)) + p
                    n_ok += 1
    open(OUT, 'wb').write(bytes(pack))
    print(f'写出 {n_ok} 帧到 {OUT}')
