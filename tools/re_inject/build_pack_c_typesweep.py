# 判别 pack C：怪类型扫描——#1841（已真机渲染的 19 字段怪帧）为基底，
# 只改 Type(181 u8) / NpcType(1566 i32) / obj-own 身份 / posA-posB x 坐标行距。
# 同宽改写=帧长与字段走查零漂移；身份用官方 0x0e00xxxx 自持族形（pack A 已证可渲染）。
# 变体：control(4/14) + Type∈{1,2,3,5}|NpcType=14 + NpcType∈{1,4,45,99}|Type=4
#   Type=2 = 玩家分支（get_scene_obj_height Lua 走 Sex/身高分支）
#   NpcType=45 = vf68 定案的静默忽略值（负对照：应当不出现）
import sys, struct
sys.path.insert(0, r'G:\pojie\ai\9yin-server-go-2433\tools\pcap_analysis')
sys.path.insert(0, r'G:\pojie\ai\9yin-server-go\internal\clientdata')

from pcaplib import build_streams, stream_frames

PCAP = r'F:/Game/Online/9yin/9yin-server-go/artifacts/npc.pcapng'
OUT = r'G:/pojie/ai/9yin-server-go/tmp/op28_analysis/npc_typesweep46.bin'
BASE_IDX = 1841

VARIANTS = [
    ('control', 4, 14),
    ('Type1',   1, 14),
    ('Type2',   2, 14),
    ('Type3',   3, 14),
    ('Type5',   5, 14),
    ('NpcT1',   4, 1),
    ('NpcT4',   4, 4),
    ('NpcT45',  4, 45),
    ('NpcT99',  4, 99),
]

def rewrite(frame, obj, own, type_val, npctype_val, dx):
    f = bytearray(frame)
    struct.pack_into('<II', f, 10, obj, own)
    # 坐标行距：posA x@+10+8、posB x@+10+0x18（同加 dx）
    for off in (18, 10 + 0x18):
        x = struct.unpack_from('<f', f, off)[0] + dx
        struct.pack_into('<f', f, off, x)
    # 字段走查改写（同宽）
    body = 10
    count = struct.unpack_from('<H', f, body + 0x3C)[0]
    off = body + 0x3E
    for _ in range(count):
        idx = struct.unpack_from('<H', f, off)[0]
        off += 2
        if idx == 181:
            f[off] = type_val & 0xFF
            off += 1
        elif idx == 1566:
            struct.pack_into('<i', f, off, npctype_val)
            off += 4
        else:
            tc = TYPES.get(idx)
            if tc == '7':
                n = struct.unpack_from('<I', f, off)[0]
                off += 4 + n
            elif tc == '1':
                off += 1
            elif tc == '2':
                off += 2
            elif tc in ('3', '5'):
                off += 4
            elif tc in ('4', '9'):
                off += 8
            else:
                raise SystemExit(f'未知线型 idx={idx} tc={tc}')
    assert off == len(f), (off, len(f))
    return bytes(f)

# 2433 表线型
TYPES = {}
for line in open(r'G:\pojie\ai\9yin-server-go\internal\clientdata\official_property_table_2433.txt',
                 encoding='utf-8-sig', errors='replace'):
    parts = line.strip().split('\t')
    if len(parts) >= 3:
        try:
            TYPES[int(parts[0])] = parts[2].strip()
        except ValueError:
            pass

base = None
for st in build_streams(PCAP):
    if not st.s2c:
        continue
    for idx, (ts, p) in enumerate(stream_frames(st.s2c)):
        if idx == BASE_IDX:
            base = p
if base is None:
    raise SystemExit('base frame not found')

pack = bytearray()
print(f'基底 #{BASE_IDX} len={len(base)}（Type=4/NpcType=14/walk）')
for i, (tag, tv, nv) in enumerate(VARIANTS):
    obj = 0x0E005A01 + i
    own = (obj & 0xFFFF) + 0xDD  # 自持族 +0xDD 相关形
    f = rewrite(base, obj, own, tv, nv, dx=3.0 * i)
    pack += struct.pack('<I', len(f)) + f
    print(f'  {tag:8s} obj={obj:#010x} own={own:#x} Type={tv} NpcType={nv} dx={3*i}m')

open(OUT, 'wb').write(bytes(pack))
print('写出', OUT, len(pack), 'bytes')
