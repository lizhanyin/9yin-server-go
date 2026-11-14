# 容错 LZ 会话解码：池 miss 时把 match 的 ln 字节从 blob 当前位置当 literal
# 脏拷继续（标记 dirty），边界越界即截断返回。用于登录簇早期帧（0x14/0x29
# 宣告-预载族）——严格 lzdec 在这些帧 pool miss 即抛。
# 用法: python -I lz_dirty_decode.py [frames.pkl] [frame# ...]
import pickle, struct, sys, os
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from lzdec import World, T

def decompress_dirty(blob, outlen, world):
    mode = blob[0]; hdr = 9 if (mode & 2) else 3
    src = hdr
    base = world.alloc(outlen)
    out = bytearray(); heap = world.heap
    flags = 1; pool = world.pool; window = -1
    tail_at = outlen - 0xb
    dirty = 0
    def reg(upto):
        nonlocal window
        while window < upto:
            pos = window + 1
            if pos + 4 <= len(out):
                v = struct.unpack_from('<I', out, pos)[0]
                pool[((v >> 12) ^ v) & 0xfff] = base + pos
            window = pos
    while True:
        write = len(out)
        if write >= tail_at:
            while write < outlen:
                if flags == 1:
                    if src + 4 > len(blob): return bytes(out), dirty
                    flags = struct.unpack_from('<I', blob, src)[0]; src += 4
                out.append(blob[src] if src < len(blob) else 0); src += 1
                write += 1; flags >>= 1
            reg(outlen - 3)
            return bytes(out), dirty
        if flags == 1:
            if src + 4 > len(blob): return bytes(out), dirty
            flags = struct.unpack_from('<I', blob, src)[0]; src += 4
        if flags & 1:
            if src + 2 > len(blob): return bytes(out), dirty
            v = struct.unpack_from('<I', blob, src)[0] if src + 4 <= len(blob) else struct.unpack_from('<H', blob, src)[0]
            idx = (v >> 4) & 0xfff; nib = v & 0xf
            flags >>= 1
            if nib:
                ln = nib + 2; src += 2
            else:
                if src + 3 > len(blob): return bytes(out), dirty
                ln = blob[src + 2]; src += 3
            srcpos = pool.get(idx)
            if srcpos is None:
                dirty += 1
                chunk = blob[src:src + ln]
                out.extend(chunk); src += ln
            else:
                out.extend(heap[srcpos:srcpos + ln])
        else:
            n = flags & 0xf; size = T[n]
            if size == 0 or src + size > len(blob): return bytes(out), dirty
            out.extend(blob[src:src + size]); src += size; flags >>= size
        reg(len(out) - 3)


def main():
    base = r'G:/pojie/ai/9yin-server-go/tmp/op28_analysis'
    src = sys.argv[1] if len(sys.argv) > 1 else base + '/frames.pkl'
    wants = set(int(a) for a in sys.argv[2:])
    frames = pickle.load(open(src, 'rb'))
    world = World()
    results = {}
    for fi, f in enumerate(frames):
        if not isinstance(f, (bytes, bytearray)) or len(f) < 12 or f[0] not in range(0x27, 0x31):
            continue
        mode = f[1]; h = 4 if (mode & 2) else 1
        if 2 + h + 4 > len(f):
            continue
        edx = struct.unpack_from('<I', f, 2 + h)[0]
        outlen = edx & (0xFFFFFFFF >> ((4 - h) * 8))
        if outlen > 0xFFFF:
            continue
        out, dirty = decompress_dirty(bytes(f[1:]), outlen, world)
        results[fi] = (f[0], out, dirty)
    print('family decoded: %d frames' % len(results))
    for fi in sorted(wants):
        if fi not in results:
            print('#%d not in family' % fi); continue
        op, out, dirty = results[fi]
        if op in (0x29, 0x2a) and len(out) >= 15:
            sub = out[0]; obj, own = struct.unpack_from('<II', out, 1)
            rec, flag2, rows = struct.unpack_from('<HHH', out, 9)
            print('#%d op=%#x sub=%d obj=%#x own=%#x rec=%d flag2=%#x rows=%d len=%d dirty=%d' % (
                fi, op, sub, obj, own, rec, flag2, rows, len(out), dirty))
            print('  head:', out[:80].hex(' '))
    return results


if __name__ == '__main__':
    main()
