# 已验证 7 帧的全字段解码（名/型/值）——对照 officialNPC0x28Frame 模板差异
import sys, struct
sys.path.insert(0, r'G:\pojie\ai\9yin-server-go-2433\tools\pcap_analysis')
sys.path.insert(0, '.')
from extract_native_npc_pack import load_types, PCAP
from pcaplib import build_streams, stream_frames

types = load_types(r'G:\pojie\ai\9yin-server-go\internal\clientdata\official_property_table_2433.txt')
names = {}
for line in open(r'G:\pojie\ai\9yin-server-go\internal\clientdata\official_property_table_2433.txt', encoding='utf-8-sig', errors='replace'):
    line = line.strip()
    if not line or line.startswith('#'):
        continue
    parts = line.split('\t')
    if len(parts) >= 2:
        try:
            names[int(parts[0])] = parts[1]
        except ValueError:
            pass

GOOD = {1841, 1898, 1956, 3542, 10472, 13176, 13271}

def decode_fields(body):
    out = []
    off = 0x3E
    while off < len(body):
        idx = struct.unpack_from('<H', body, off)[0]
        off += 2
        tc = types.get(idx)
        name = names.get(idx, f'?{idx}')
        if tc == '1':
            out.append((idx, name, 'u8', body[off])); off += 1
        elif tc == '2':
            out.append((idx, name, 'u16', struct.unpack_from('<H', body, off)[0])); off += 2
        elif tc == '3':
            out.append((idx, name, 'i32', struct.unpack_from('<i', body, off)[0])); off += 4
        elif tc == '4':
            out.append((idx, name, 'i64', struct.unpack_from('<q', body, off)[0])); off += 8
        elif tc == '5':
            out.append((idx, name, 'f32', round(struct.unpack_from('<f', body, off)[0], 4))); off += 4
        elif tc == '7':
            n = struct.unpack_from('<I', body, off)[0]
            off += 4
            raw = body[off:off+n]
            out.append((idx, name, 'str', raw[:-1].decode('gbk', errors='replace')))
            off += n
        elif tc == '8':
            n = struct.unpack_from('<I', body, off)[0]
            off += 4
            out.append((idx, name, 'wstr', body[off:off+n].hex()))
            off += n
        elif tc == '9':
            out.append((idx, name, 'obj', hex(struct.unpack_from('<Q', body, off)[0]))); off += 8
    return out

for st in build_streams(PCAP):
    if not st.s2c:
        continue
    for idx, (ts, p) in enumerate(stream_frames(st.s2c)):
        if idx not in GOOD:
            continue
        obj, own = struct.unpack_from('<II', p, 10)
        body = p[10:]
        x, y, z, yaw = struct.unpack_from('<4f', body, 8)
        bx, by, bz, byaw = struct.unpack_from('<4f', body, 0x18)
        scale = struct.unpack_from('<f', body, 0x28)[0]
        flag = struct.unpack_from('<I', body, 0x38)[0]
        print(f'===== #{idx} obj={obj:#x} own={own:#x} posA=({x:.1f},{y:.1f},{z:.1f},{yaw:.3f}) posB=({bx:.1f},{by:.1f},{bz:.1f},{byaw:.3f}) scale={scale} flag={flag}')
        for fidx, fname, ftype, fval in decode_fields(body):
            print(f'  {fidx:<5d} {fname:16s} {ftype:4s} {fval}')
        print()
