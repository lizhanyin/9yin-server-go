# -*- coding: utf-8 -*-
"""外服抓包 0x47 族（op27/28/29/2A/30）LZ 全会话解码 + op29 记录批行解。

用法: python -I op29_batch_decode.py <pcapng> [--hex]
输出: 每帧 op/解压长度/明文摘要；op29 批量按
  [0]=op [1]=mode [2:6]=len-1 [6:10]=outlen ... 记录块（解压后）
  行块方言: [u16 idx][值]（串=[u8 len=strlen+1][01 00][值+NUL]）
"""
import os
import struct
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from lzdec import World, decompress, DecodeError  # noqa: E402

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'pcap_analysis'))
from pcaplib import build_streams, stream_frames  # noqa: E402

FAMILY = set(range(0x27, 0x31))


def frame_decode(p, world):
    mode = p[1]
    hdrlen = 4 if (mode & 2) else 1
    edx = struct.unpack_from('<I', p, 2 + hdrlen)[0]
    outlen = edx & (0xFFFFFFFF >> ((4 - hdrlen) * 8))
    if outlen > 0xFFFF:
        raise DecodeError('outlen %#x > 0xffff' % outlen)
    return decompress(p[1:], outlen, world), outlen


def parse_record_block(data, off=0):
    """解压后记录块: [u16 idx][值] 流。返回 [(idx, value)] 与耗尽偏移。"""
    out = []
    n = len(data)
    while off + 2 <= n:
        idx = struct.unpack_from('<H', data, off)[0]
        off += 2
        if off >= n:
            break
        b0 = data[off]
        if b0 in (0, 1, 2, 3, 4):  # 小整型短形
            out.append((idx, b0))
            off += 1
        elif b0 == 0x0c or b0 == 0x0d:  # i32/i64 变体（0x0c=i32? 0x0d=i64?）
            ln = 4 if b0 == 0x0c else 8
            v = int.from_bytes(data[off + 1:off + 1 + ln], 'little', signed=True)
            out.append((idx, v))
            off += 1 + ln
        else:
            # 串形: [u8 len][01 00][len 字节]
            ln = b0
            if data[off + 1:off + 3] == b'\x01\x00' and off + 3 + ln <= n:
                out.append((idx, data[off + 3:off + 3 + ln].rstrip(b'\x00').decode('gbk', 'replace')))
                off += 3 + ln
            else:
                out.append((idx, '<unknown %02x @%d>' % (b0, off)))
                break
    return out, off


def main():
    sys.stdout.reconfigure(encoding='utf-8', errors='replace')
    path = sys.argv[1]
    show_hex = '--hex' in sys.argv
    for st in build_streams(path):
        if st.key[3] != 2001:
            continue
        frames = [(ts, fr) for ts, fr in stream_frames(st.s2c)]
        t0 = frames[0][0]
        world = World()
        ok = fail = 0
        for ts, fr in frames:
            if not fr or fr[0] not in FAMILY:
                continue
            try:
                out, outlen = frame_decode(fr, world)
                ok += 1
            except (DecodeError, struct.error) as e:
                fail += 1
                print('[%7.2fs] op%02X len=%d 解压失败: %s' % ((ts - t0) / 1e6, fr[0], len(fr), e))
                continue
            if fr[0] != 0x29:
                continue
            d = bytes(out)
            print('[%7.2fs] op29 len=%d → 解压 %dB' % ((ts - t0) / 1e6, len(fr), len(d)))
            if show_hex:
                for o in range(0, min(len(d), 0x180), 16):
                    ch = d[o:o + 16]
                    print('  %04x: %-48s %s' % (o, ' '.join('%02x' % b for b in ch),
                                                ''.join(chr(b) if 32 <= b < 127 else '.' for b in ch)))
            # 记录批明文头: [15:19]obj [19:23]owner [23:27]rec [27:29]rows
            if len(d) >= 29:
                obj, owner = struct.unpack_from('<II', d, 15)
                rec = struct.unpack_from('<I', d, 23)[0]
                rows = struct.unpack_from('<H', d, 27)[0]
                print('    obj=%#x owner=%#x rec=%d rows=%d' % (obj, owner, rec, rows))
                cells, end = parse_record_block(d, 29)
                for idx, v in cells:
                    print('    c%-3d = %r' % (idx, v))
                if end < len(d):
                    print('    (剩 %dB 未解: %s)' % (len(d) - end, d[end:end + 32].hex()))
        print('LZ: ok=%d fail=%d' % (ok, fail))
        break


if __name__ == '__main__':
    main()
