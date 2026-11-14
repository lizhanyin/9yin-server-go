# -*- coding: utf-8 -*-
"""初始账号.pcapng（外服，只完成初始教程关卡）方言探针。

产出：
  1. 全时间线 dump（op 级 + custommsg 级）→ stdout / 可选文件
  2. 聚合统计：op 计数、custommsg 计数、记录表(0x11/12/13)流量按 rec 分组
  3. 任务链深挖：rec 29/30/31/33 行加/格写、StepEventDataRecord(377)、
     rec_newteacher(207)、OpenMapRec(51) 位图

用法：python -I fresh_tutorial_probe.py <pcapng> [--full]
"""
import os
import struct
import sys
from collections import Counter, defaultdict

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(HERE, 'pcap_analysis'))
from pcaplib import build_streams, stream_frames  # noqa: E402
import decode  # noqa: E402

RECBIN = os.path.join(HERE, '..', 'internal', 'clientdata', 'official_record_table.bin')


def load_rec_names():
    d = open(RECBIN, 'rb').read()
    cnt = struct.unpack_from('<H', d, 1)[0]
    off = 3
    names = {}
    for i in range(cnt):
        z = d.find(b'\x00', off)
        names[i] = d[off:z].decode('gbk', 'replace')
        ncols = struct.unpack_from('<H', d, z + 1)[0]
        off = z + 3 + ncols
    return names


# 记录表列 schema（供 0x11 行加解码：列型按官方 447 表；bin 里每列 1 字节类型码，
# 无列名——1=byte 2=u16 3=i32 4=i64 6=f64 7=窄串 8=宽串 9=对象引用）
def load_rec_schema():
    d = open(RECBIN, 'rb').read()
    cnt = struct.unpack_from('<H', d, 1)[0]
    off = 3
    schema = {}
    for i in range(cnt):
        z = d.find(b'\x00', off)
        ncols = struct.unpack_from('<H', d, z + 1)[0]
        off2 = z + 3
        schema[i] = [('c%d' % c, t) for c, t in enumerate(d[off2:off2 + ncols])]
        off = off2 + ncols
    return schema


SCHEMA = load_rec_schema()


def read_cell(fr, off, t):
    n = len(fr)
    try:
        if t == 1:
            return fr[off], off + 1
        if t == 2:
            return struct.unpack_from('<H', fr, off)[0], off + 2
        if t == 3:
            return struct.unpack_from('<i', fr, off)[0], off + 4
        if t == 4:
            return struct.unpack_from('<q', fr, off)[0], off + 8
        if t == 5:
            return round(struct.unpack_from('<f', fr, off)[0], 2), off + 4
        if t == 6:
            return struct.unpack_from('<d', fr, off)[0], off + 8
        if t == 7:
            ln = struct.unpack_from('<I', fr, off)[0]
            off += 4
            s = fr[off:off + ln].rstrip(b'\x00').decode('gbk', 'replace')
            return s, off + ln
        if t == 8:
            ln = struct.unpack_from('<I', fr, off)[0]
            off += 4
            s = fr[off:off + ln * 2].decode('utf-16-le', 'replace')
            return s, off + ln * 2
        if t == 9:
            oid, owner = struct.unpack_from('<II', fr, off)
            return 'obj(%#x,%#x)' % (oid, owner), off + 8
    except (struct.error, IndexError):
        return '<溢出>', off
    return None, off


def cell_str(v):
    if isinstance(v, float):
        return '%g' % v
    if isinstance(v, str):
        return '"%s"' % v[:48]
    return str(v)


def decode_op11_rows(fr):
    """0x11 行加：obj@2 owner@6 rec@10 起始行@12 行数@14 数据@16。按 schema 逐列解。"""
    if len(fr) < 16:
        return None
    obj, owner, rec, start_row, nrows = struct.unpack_from('<IIHHH', fr, 2)
    cols = SCHEMA.get(rec)
    if not cols:
        return None
    rows = []
    off = 16
    for r in range(nrows):
        vals = []
        for nm, t in cols:
            v, off = read_cell(fr, off, t)
            vals.append('%s=%s' % (nm, cell_str(v)))
        rows.append(vals)
        if off > len(fr):
            break
    return rec, start_row, rows, off, len(fr)


def main():
    sys.stdout.reconfigure(encoding='utf-8', errors='replace')
    path = sys.argv[1]
    full = '--full' in sys.argv
    decode.REC_NAMES = load_rec_names()

    streams = build_streams(path)
    events = []  # (t, dir, desc, sig, frame)
    for st in streams:
        if st.key[3] in (4000,):  # svrlist
            continue
        tbl = decode.Table09()
        s2c_frames = stream_frames(st.s2c)
        for ts, fr in s2c_frames:
            if fr and fr[0] == 0x09:
                tbl.feed(fr)
        for ts, fr in s2c_frames:
            d, s = decode.describe_s2c(fr, tbl)
            events.append((ts, 'S2C', d, s, fr))
        for ts, fr in stream_frames(st.c2s):
            d, s = decode.describe_c2s(fr)
            events.append((ts, 'C2S', d, s, fr))
    t0 = events[0][0] if events else 0
    events.sort(key=lambda e: e[0])

    # ---- 聚合 ----
    op_c = Counter()
    msg_c = Counter()
    rec_traffic = defaultdict(lambda: Counter())  # rec -> action counter
    views = []
    for t, dr, d, s, fr in events:
        sig = s.split(':', 1)[1] if ':' in s else s
        if dr == 'S2C':
            op_c['s2c op%02X' % fr[0]] += 1
            if s.startswith('S2C:m'):
                msg_c['s2c %s' % s.split(':', 1)[1].split(':', 1)[0]] += 1
            if fr[0] == 0x11 and len(fr) >= 16:
                rec_traffic[struct.unpack_from('<H', fr, 10)[0]]['+行'] += 1
            if fr[0] == 0x12 and len(fr) >= 14:
                rec_traffic[struct.unpack_from('<H', fr, 10)[0]]['-行'] += 1
            if fr[0] == 0x13 and len(fr) >= 17:
                rec_traffic[struct.unpack_from('<H', fr, 10)[0]]['格写'] += 1
            if fr[0] == 0x15 and len(fr) >= 7:
                views.append(struct.unpack_from('<H', fr, 1)[0])
        else:
            op_c['c2s op%02X' % fr[0]] += 1
            if s.startswith('C2S:m'):
                msg_c['c2s %s' % s.split(':', 1)[1].split(':', 1)[0]] += 1

    print('=' * 100)
    print('OP 计数（前 40）')
    for k, v in op_c.most_common(40):
        print('  %-16s %d' % (k, v))
    print('=' * 100)
    print('custommsg 计数')
    for k, v in sorted(msg_c.items()):
        print('  %-24s %d' % (k, v))
    print('=' * 100)
    print('记录表流量（rec: 名字 [动作×次数]）')
    for rec in sorted(rec_traffic):
        c = rec_traffic[rec]
        print('  #%-3d %-28s %s' % (rec, decode.REC_NAMES.get(rec, '?'),
                                    ' '.join('%s×%d' % kv for kv in sorted(c.items()))))
    print('=' * 100)
    print('视图建立序（0x15）:', views)

    # ---- 任务链深挖：rec 29/30/31/33/377/207/51 的 0x11 行内容 ----
    print('=' * 100)
    print('任务/教程相关 0x11 行加内容')
    for t, dr, d, s, fr in events:
        if dr == 'S2C' and fr[0] == 0x11:
            r = decode_op11_rows(fr)
            if not r:
                continue
            rec, start_row, rows, off, ln = r
            if rec in (29, 30, 31, 33, 377, 207, 51, 58, 94, 102, 122, 168):
                print('  [%7.2fs] rec#%d %s 行%d起+%d行 (尾部耗 %d/%dB):'
                      % ((t - t0) / 1e6, rec, decode.REC_NAMES.get(rec), start_row, len(rows), off, ln))
                for i, row in enumerate(rows):
                    print('      行%d: %s' % (start_row + i, '; '.join(row)))
    print('=' * 100)
    print('任务/教程相关 0x13 格写（rec 29/30/31/33/377/207/51）')
    for t, dr, d, s, fr in events:
        if dr == 'S2C' and fr[0] == 0x13 and len(fr) >= 17:
            rec = struct.unpack_from('<H', fr, 10)[0]
            if rec in (29, 30, 31, 33, 377, 207, 51, 58, 94):
                col = fr[16]
                cols = SCHEMA.get(rec, [])
                cn = cols[col][0] if col < len(cols) else '?col%d' % col
                ct = cols[col][1] if col < len(cols) else 3
                v, _ = read_cell(fr, 17, ct)
                print('  [%7.2fs] rec#%d %s 行%d 列%d(%s) = %s'
                      % ((t - t0) / 1e6, rec, decode.REC_NAMES.get(rec),
                         struct.unpack_from('<H', fr, 14)[0], col, cn, cell_str(v)))
    print('=' * 100)
    if full:
        print('全时间线（%d 事件）' % len(events))
        last_sig = None
        run = 0
        for t, dr, d, s, fr in events:
            ts = (t - t0) / 1e6
            if s == last_sig:
                run += 1
                if run == 2:
                    print('  ...')
                if run > 2:
                    continue
            else:
                run = 1
                last_sig = s
            print('  [%8.2fs] %-3s %s' % (ts, dr, d))


if __name__ == '__main__':
    main()
