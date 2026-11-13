# -*- coding: utf-8 -*-
"""初始账号.pcapng 视图行解码（用抓包自带 0x09 表，2086 条第三方言）。

用法: python -I 初始账号_viewrows.py <pcapng> [view ...]   默认 40 43 46 2 1 41
"""
import os
import struct
import sys

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), 'pcap_analysis'))
from pcaplib import build_streams, stream_frames  # noqa: E402


def read_val(fr, off, t):
    n = len(fr)
    if t == 1:
        return fr[off], off + 1
    if t == 2:
        return struct.unpack_from('<H', fr, off)[0], off + 2
    if t == 3:
        return struct.unpack_from('<i', fr, off)[0], off + 4
    if t == 4:
        return struct.unpack_from('<q', fr, off)[0], off + 8
    if t == 5:
        return round(struct.unpack_from('<f', fr, off)[0], 3), off + 4
    if t == 6:
        return round(struct.unpack_from('<d', fr, off)[0], 3), off + 8
    if t == 7:
        ln = struct.unpack_from('<I', fr, off)[0]
        off += 4
        return fr[off:off + ln].rstrip(b'\x00').decode('gbk', 'replace'), off + ln
    if t == 8:
        ln = struct.unpack_from('<I', fr, off)[0]  # 字节数（pcaplib tag7 同义）
        off += 4
        return fr[off:off + ln].decode('utf-16-le', 'replace').rstrip('\x00'), off + ln
    raise ValueError('bad type %d' % t)


def main():
    sys.stdout.reconfigure(encoding='utf-8', errors='replace')
    path = sys.argv[1]
    want = [int(v) for v in sys.argv[2:]] or [40, 43, 46, 2, 1, 41]
    tbl = {}
    for st in build_streams(path):
        if st.key[3] != 2001:
            continue
        frames = [(ts, fr) for ts, fr in stream_frames(st.s2c)]
        for ts, fr in frames:
            if fr and fr[0] == 0x09:
                cnt = struct.unpack_from('<H', fr, 1)[0]
                off = 3
                for i in range(cnt):
                    z = fr.find(b'\x00', off)
                    tbl[i] = (fr[off:z].decode('gbk', 'replace'), fr[z + 1])
                    off = z + 2
                break
        t0 = frames[0][0]
        for ts, fr in frames:
            if fr[0] != 0x18 or len(fr) < 7:
                continue
            view, objidx, cnt = struct.unpack_from('<HHH', fr, 1)
            if view not in want:
                continue
            off = 7
            cells = []
            try:
                for _ in range(cnt):
                    idx = struct.unpack_from('<H', fr, off)[0]
                    off += 2
                    nm, t = tbl.get(idx, ('?%d' % idx, 3))
                    v, off = read_val(fr, off, t)
                    cells.append('%s=%r' % (nm, v))
            except (struct.error, ValueError, IndexError) as e:
                cells.append('<解析中断@%d: %s>' % (off, e))
            print('[%6.2fs] v%d obj#%d (%dB): %s' % ((ts - t0) / 1e6, view, objidx, len(fr), '; '.join(cells)))
        break


if __name__ == '__main__':
    main()
