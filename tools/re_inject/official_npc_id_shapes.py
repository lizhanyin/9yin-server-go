# 官方 npc.pcapng 0x28/0x0D 帧对象身份普查：obj/own 实值形状
# （wire id 形状判别：官方 NPC obj u32 高位段 vs 我方 0x11000xxx 序列）
import sys, struct
sys.path.insert(0, r'G:\pojie\ai\9yin-server-go-2433\tools\pcap_analysis')
from pcaplib import build_streams, stream_frames

PCAP = r'F:/Game/Online/9yin/9yin-server-go/artifacts/npc.pcapng'
streams = build_streams(PCAP)

rows = []
for st in streams:
    if not st.s2c:
        continue
    try:
        frames = stream_frames(st.s2c)
    except Exception as e:
        print('stream err', e)
        continue
    for idx, (ts, p) in enumerate(frames):
        if not p:
            continue
        op = p[0]
        if op == 0x0D and len(p) >= 0x3F:
            obj, own = struct.unpack_from('<II', p, 1)
            rows.append(('0x0D', idx, obj, own, len(p)))
        elif op == 0x28 and len(p) >= 10:
            mode = p[1]
            outlen = struct.unpack_from('<I', p, 6)[0]
            if mode == 0x46 and len(p) >= 10 + 8 and outlen == len(p) - 10:
                body = p[10:]
                obj, own = struct.unpack_from('<II', body, 0)
                rows.append(('0x28/46', idx, obj, own, len(p)))

print(f'{len(rows)} object frames')
seen = set()
for kind, idx, obj, own, ln in rows:
    key = (kind, obj, own)
    if key in seen:
        continue
    seen.add(key)
    mo = (obj >> 8) & 0xFFFFF
    print(f'{kind:8s} #{idx:<5d} obj={obj:#010x} own={own:#010x} len={ln:<4d} motion_id=(obj>>8)&0xFFFFF={mo:#x}({mo})')
