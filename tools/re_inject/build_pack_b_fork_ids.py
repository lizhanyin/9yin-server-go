# 判别 pack B：官方 19 字段怪帧（#1841/#1898/#1956）obj/own 改写为 fork 形
#（0x11001xxx 序列），其余字节零改写——隔离「id 形状」这最后一个变量。
# pack A（原帧）已真机渲染 ✓ → 若 B 也渲染=fork id 可用；B 不渲染=id 借用必要。
import sys, struct
sys.path.insert(0, r'G:\pojie\ai\9yin-server-go-2433\tools\pcap_analysis')
from pcaplib import build_streams, stream_frames

PCAP = r'F:/Game/Online/9yin/9yin-server-go/artifacts/npc.pcapng'
OUT = r'G:/pojie/ai/9yin-server-go/tmp/op28_analysis/npc_forkid46_replay.bin'
GOOD = {1841, 1898, 1956}  # 19 字段怪形（无 BufferInfo 内嵌身份串，改写干净）

# fork 形身份：避开我方场景 NPC 已用段（0x11000005..0103），取 0x11001001 起
FORK_IDS = [(0x11001001, 0x00003401), (0x11001002, 0x00003402), (0x11001003, 0x00003403)]

frames = []
for st in build_streams(PCAP):
    if not st.s2c:
        continue
    for idx, (ts, p) in enumerate(stream_frames(st.s2c)):
        if idx in GOOD:
            frames.append((idx, p))
frames.sort()

pack = bytearray()
for i, (idx, p) in enumerate(frames):
    f = bytearray(p)
    obj, own = FORK_IDS[i]
    struct.pack_into('<II', f, 10, obj, own)  # 体头 obj@0/own@4 = 帧 +10
    pack += struct.pack('<I', len(f)) + bytes(f)
    print(f'#{idx} obj={obj:#x} own={own:#x} len={len(f)}')

open(OUT, 'wb').write(bytes(pack))
print('写出', OUT, len(pack), 'bytes')
