# 四包 configID→official Resource/id 提取（字节搜索法，免走查）:
#   name@0 u32、ConfigID=idx 1447 (b7 00 in 0x0D frame / a7 05 in body)、Resource=idx 1448 (b8 00 / a8 05)
# 用法: python -I extract_resource_map.py
import pickle, struct, re, sys, os
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from lzdec import World, DecodeError

BASE = r'G:/pojie/ai/9yin-server-go/tmp/op28_analysis'
FAMILY = set(range(0x27, 0x31))

def frame_decode(p, world):
    mode = p[1]
    hdrlen = 4 if (mode & 2) else 1
    if 2 + hdrlen + 4 > len(p): raise DecodeError('short')
    edx = struct.unpack_from('<I', p, 2 + hdrlen)[0]
    outlen = edx & (0xFFFFFFFF >> ((4 - hdrlen) * 8))
    if outlen > 0xFFFF: raise DecodeError('outlen')
    from lzdec import decompress
    return decompress(p[1:], outlen, world), outlen

def grab(body, base):
    """base: body 内值对搜索（idx 已含）。返回 (configID, resource)。"""
    out = [None, None]
    for tag, which in ((b'\xa7\x05', 0), (b'\xa8\x05', 1)):
        hit = re.search(re.escape(tag) + rb'([\x02-\x40])\x00\x00\x00', body)
        if not hit: continue
        ln = hit.group(1)[0]
        off = hit.start() + 6
        if off + ln <= len(body) and 3 < ln < 48:
            raw = body[off:off+ln]
            if raw.endswith(b'\x00'):
                out[which] = raw[:-1].decode('gbk', 'replace')
    return out

m = {}  # configID -> (resource, name, src)
def add(cid, res, name, src):
    if not cid: return
    prev = m.get(cid)
    if prev is None or (res and not prev[0]):
        m[cid] = (res, name, src)

# 1) l.pcapng 已解压体
for idx, ol, p in pickle.load(open(BASE + '/decoded_28.pkl', 'rb')):
    name = struct.unpack_from('<I', p, 0)[0]
    cid, res = grab(p, None)
    add(cid, res, name, 'l')
print('after l:', len(m))

# 2) c.pcapng (frames.pkl) 会话解码
frames = pickle.load(open(BASE + '/frames.pkl', 'rb'))
world = World(); ok = 0
for f in frames:
    if not isinstance(f, (bytes, bytearray)) or len(f) < 10 or f[0] not in FAMILY: continue
    try:
        out, _ = frame_decode(bytes(f), world); ok += 1
    except DecodeError: continue
    if f[0] != 0x28: continue
    name = struct.unpack_from('<I', out, 0)[0]
    cid, res = grab(out, None)
    add(cid, res, name, 'c')
print('c family ok=%d, after c:' % ok, len(m))
pickle.dump(m, open(BASE + '/cid_resource_map.pkl', 'wb'))
for cid in sorted(m)[:10]:
    res, name, src = m[cid]
    print('  %-28s %-24s %#010x %s' % (cid, res, name, src))
