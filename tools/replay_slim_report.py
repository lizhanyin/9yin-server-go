# -*- coding: utf-8 -*-
"""replay_p1_slim.bin / replay_p2_slim.bin 内容盘点（B 线回放 bin 瘦身后审计）。

打包格式 = [u32 len][frame]×N（replayFramesPack 逐帧回放）。
P1 = Phase A（0x0B+化身 0x28+首爆发），P2 = Phase B（第二爆发+物化簇）。

解码链：0x27-0x30 压缩族走 World2 池 LZ 解压（P1 先 P2 后，按回放顺序
喂池）；0x28 明文按 AddObject 布局（obj@0/own@4/13f@8/cnt@60）拆字段；
0x1E custom 取 u16 子号@1；0x0D/0x1F 按 obj/own 定位。

Usage: python -I tools/replay_slim_report.py
"""
import struct
import sys
from collections import Counter

sys.stdout.reconfigure(encoding='utf-8')
sys.path.insert(0, r'G:\pojie\ai\9yin-server-go\tools\pcap_analysis')
sys.path.insert(0, r'G:\pojie\ai\9yin-server-go\tools')
import lzdec  # noqa: E402
from op46_profile_stitch import World2  # noqa: E402

OUT = r'G:\pojie\ai\9yin-server-go\tmp\op28_analysis'
FAMILY = set(range(0x27, 0x31))


def unpack_bin(path):
    raw = open(path, 'rb').read()
    frames, rest, bad = [], raw, 0
    while len(rest) >= 4:
        fl = struct.unpack_from('<I', rest)[0]
        if fl <= 0 or fl > len(rest) - 4 or fl > 0x10000:
            bad += 1
            break
        frames.append(bytes(rest[4:4 + fl]))
        rest = rest[4 + fl:]
    return frames, len(rest), bad


def frame_decode(p, world):
    mode = p[1]
    hdrlen = 4 if (mode & 2) else 1
    outlen = struct.unpack_from('<I', p, 2 + hdrlen)[0] & (0xFFFFFFFF >> ((4 - hdrlen) * 8))
    return lzdec.decompress(p[1:], outlen, world), outlen


def advance(world, frame):
    """单次池推进：压缩族解压入池（失败也 alloc 保底），其余明文滑窗注册。
    返回 (plaintext, outlen) 供 0x28 走查，其他恒 (None, None)。"""
    if frame[0] in FAMILY:
        try:
            return frame_decode(frame, world)
        except lzdec.DecodeError:
            world.alloc(0)
            return None, None
    world.register_raw(frame)
    return None, None


def describe(idx, frame, pl, outlen):
    op, n = frame[0], len(frame)
    if op == 0x0B:
        u32s = struct.unpack_from('<4I', frame, 1) if n >= 17 else ()
        return f'#{idx:3d} 0x0B len={n} u32@1={[hex(u) for u in u32s]}'
    if op == 0x0D:
        obj, own = struct.unpack_from('<II', frame, 0)
        return f'#{idx:3d} 0x0D len={n} obj={obj:08x} own={own:08x} head={frame[:32].hex()}'
    if op == 0x1F:
        obj, own = struct.unpack_from('<II', frame, 1)
        extra = ''
        if n >= 25:
            x, y, z, yaw = struct.unpack_from('<4f', frame, 9)
            extra = f' pos=({x:.1f},{y:.1f},{z:.1f}) yaw={yaw:.2f}'
        return f'#{idx:3d} 0x1F len={n} obj={obj:08x} own={own:08x}{extra}'
    if op == 0x28:
        if pl is None:
            return f'#{idx:3d} 0x28 wire={n} LZ-FAIL'
        obj, own = struct.unpack_from('<II', pl, 0)
        if outlen >= 62:
            f = struct.unpack_from('<13f', pl, 8)
            cnt = struct.unpack_from('<H', pl, 60)[0]
            return (f'#{idx:3d} 0x28 wire={n} out={outlen} obj={obj:08x} own={own:08x} cnt={cnt} '
                    f'pos=({f[0]:.1f},{f[1]:.1f},{f[2]:.1f}) yaw={f[3]:.1f} scale={f[6]:.2f}')
        return f'#{idx:3d} 0x28 wire={n} out={outlen} obj={obj:08x} own={own:08x} pl[:32]={pl[:32].hex()}'
    if op == 0x1E:
        mid = struct.unpack_from('<H', frame, 1)[0] if n >= 3 else -1
        return f'#{idx:3d} 0x1E len={n} m{mid}'
    if op == 0x2A:
        obj, = struct.unpack_from('<I', frame, 1) if n >= 5 else (0,)
        return f'#{idx:3d} 0x2A len={n} obj@1={obj:08x}'
    return f'#{idx:3d} {op:02X} len={n}'


def report(path, title, world, detail_pred=lambda i, f: True, limit=None):
    frames, rest, bad = unpack_bin(path)
    total = 4 * len(frames) + sum(len(f) for f in frames)
    print(f'\n===== {title}\n  {path}')
    print(f'  frames={len(frames)} packed_bytes={total} trailing={rest} badlen={bad}')
    hist = Counter(f[0] for f in frames)
    for op, c in sorted(hist.items()):
        sizes = [len(f) for f in frames if f[0] == op]
        print(f'  op 0x{op:02X}: {c:4d} 帧  {sum(sizes):6d} B  min={min(sizes):5d} max={max(sizes):5d}')
    customs = Counter(struct.unpack_from('<H', f, 1)[0] for f in frames if f[0] == 0x1E and len(f) >= 3)
    if customs:
        print('  0x1E 子号:', ', '.join(f'm{m}×{c}' for m, c in customs.most_common(20)))
    shown = 0
    for i, f in enumerate(frames):
        pl, outlen = advance(world, f)
        if not detail_pred(i, f):
            continue
        print('  ' + describe(i, f, pl, outlen))
        shown += 1
        if limit and shown >= limit:
            print(f'  …（详情止于 #{i}，后续 {len(frames) - i - 1} 帧仅推进池）')
    return frames


def main():
    world = World2()
    report(f'{OUT}\\replay_p1_slim.bin', 'P1 (Phase A 进场帧)', world,
           detail_pred=lambda i, f: True)
    report(f'{OUT}\\replay_p2_slim.bin', 'P2 (Phase B 第二爆发+物化簇)', world,
           detail_pred=lambda i, f: f[0] in (0x0B, 0x0D, 0x1F, 0x28, 0x2A),
           limit=200)




# ---------- 增强：压缩族内层 + 样本帧 + 原始抓包喂池解化身 ----------

def enhanced(world_p1, world_p2):
    import os
    CAP = r'F:\Game\Online\9yin\9yin-server-go\artifacts\c.pcapng'
    print('\n===== 压缩族内层识别（本池尽力解压）')
    for tag, frames in (('P1', unpack_bin(OUT + '/replay_p1_slim.bin')[0]),
                        ('P2', unpack_bin(OUT + '/replay_p2_slim.bin')[0])):
        w = World2()
        for i, f in enumerate(frames):
            if f[0] in FAMILY:
                try:
                    pl, outlen = frame_decode(f, w)
                    inner = f'inner=0x{pl[0]:02X}' if pl else ''
                    extra = ''
                    if pl and pl[0] == 0x1E and outlen >= 3:
                        extra = f' m{struct.unpack_from("<H", pl, 1)[0]}'
                    print(f'  {tag} #{i:3d} 0x{f[0]:02X} wire={len(f)} out={outlen} {inner}{extra}'
                          f' pl[:24]={pl[:24].hex()}')
                except lzdec.DecodeError:
                    print(f'  {tag} #{i:3d} 0x{f[0]:02X} wire={len(f)} LZ-FAIL')
                # 池已在 frame_decode 内推进
            else:
                w.register_raw(f)

    print('\n===== 样本帧 hex（0x10/0x15/0x38/0x0B）')
    for tag, frames in (('P1', unpack_bin(OUT + '/replay_p1_slim.bin')[0]),
                        ('P2', unpack_bin(OUT + '/replay_p2_slim.bin')[0])):
        for want in (0x10, 0x15, 0x38, 0x0B):
            got = [(i, f) for i, f in enumerate(frames) if f[0] == want][:3]
            for i, f in got:
                print(f'  {tag} #{i:3d} 0x{want:02X} len={len(f)} {f.hex()}')

    if not os.path.exists(CAP):
        print(f'\n原始抓包不可达，跳过化身交叉解码: {CAP}')
        return
    print('\n===== 原始 c.pcapng 喂池交叉解码（slim 帧在原流逐字节定位，按原流前缀喂池）')
    import importlib.util
    spec = importlib.util.spec_from_file_location(
        'stitch', r'G:\pojie\ai\9yin-server-go\tools\op46_profile_stitch.py')
    stitch = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(stitch)
    conns = stitch.decode_capture(CAP)
    cap = [bytes(f) for f in max(conns, key=len)]
    print(f'  原流帧数={len(cap)}')

    def lz(fr, w):
        return stitch.lz_decode(fr, w)
    slim1 = unpack_bin(OUT + '/replay_p1_slim.bin')[0]
    slim2 = unpack_bin(OUT + '/replay_p2_slim.bin')[0]
    # 逐 slim 帧在 cap 中按序定位（passthrough 应逐字节相等）
    pos = []
    cur = 0
    for tag, frames in (('P1', slim1), ('P2', slim2)):
        row = []
        for f in frames:
            hit = next((j for j in range(cur, len(cap)) if cap[j] == f), None)
            row.append(hit)
            if hit is not None:
                cur = hit + 1
        pos.append((tag, row))
        miss = [i for i, h in enumerate(row) if h is None]
        print(f'  {tag}: 定位 {len(row)-len(miss)}/{len(row)}；未命中 idx={miss[:12]}'
              f'；首尾序位={row[0]}..{row[-1] if row[-1] is not None else "?"}')
    # 全部压缩族帧按原流前缀喂池解码
    world = World2()
    fed_upto = 0

    def feed_prefix(j):
        nonlocal fed_upto
        for k in range(fed_upto, j):
            f = cap[k]
            if f[0] in FAMILY:
                try:
                    lz(f, world)
                except Exception:
                    world.alloc(0)
            else:
                world.register_raw(f)
        fed_upto = max(fed_upto, j)

    for tag, frames in (('P1', slim1), ('P2', slim2)):
        row = dict((('P1', pos[0][1]), ('P2', pos[1][1])))[tag]
        for i, f in enumerate(frames):
            if f[0] not in FAMILY:
                continue
            hit = row[i]
            if hit is None:
                print(f'  {tag} #{i:3d} 0x{f[0]:02X} wire={len(f)} 原流未定位')
                continue
            feed_prefix(hit)
            try:
                pl, outlen = lz(f, world)
            except lzdec.DecodeError as e:
                print(f'  {tag} #{i:3d} 0x{f[0]:02X} cap#{hit} wire={len(f)} 交叉解码仍失败: {e}')
                continue
            note = ''
            if f[0] in (0x28, 0x2A):
                obj, own = struct.unpack_from('<II', pl, 0)
                note = f' obj={obj:08x} own={own:08x}'
                if outlen >= 62 and f[0] == 0x28:
                    fl = struct.unpack_from('<13f', pl, 8)
                    note += (f' pos=({fl[0]:.1f},{fl[1]:.1f},{fl[2]:.1f}) yaw={fl[3]:.1f}'
                             f' scale={fl[6]:.2f} cnt={struct.unpack_from("<H", pl, 60)[0]}')
            txt = pl[:40].hex()
            print(f'  {tag} #{i:3d} 0x{f[0]:02X} cap#{hit} wire={len(f)} out={outlen}{note} pl[:40]={txt}')
            if f[0] == 0x28 and outlen > 1000:
                fn = OUT + '/p1_avatar_0x28_plaintext.bin'
                open(fn, 'wb').write(pl)
                print(f'    化身明文已存 {fn}')


if __name__ == '__main__':
    main()
    enhanced(None, None)
