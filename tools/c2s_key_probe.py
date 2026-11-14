# -*- coding: utf-8 -*-
"""c.pcapng C2S 传输 key/相位探针。

已知 S2C 用 KEY=CC3B6DFA 帧内 i%4 XOR + EE 转义可解；C2S 疑似方言不同。
对照 npc.pcapng（C2S 已知可解，同库同 key）。

Usage: python -I c2s_key_probe.py <cap.pcapng> [--dump N]
"""
import struct
import sys
from collections import Counter

sys.path.insert(0, r'G:\pojie\ai\9yin-server-go-2433\tools\pcap_analysis')
import pcaplib  # noqa: E402

KEY = pcaplib.KEY
KNOWN_C2S_OPS = {0x02, 0x04, 0x07, 0x09, 0x0A, 0x0D, 0x18, 0x1E, 0x1F, 0x2D}


def raw_frames(chunks):
    """EE EE 切帧+去转义，但【不做 XOR】，返回 raw body 列表（相位事实保留）。"""
    frames, body = [], bytearray()
    for _, payload in chunks:
        i, ln = 0, len(payload)
        while i < ln:
            b = payload[i]
            if b == 0xEE and i + 1 < ln:
                nxt = payload[i + 1]
                if nxt == 0x00:
                    body.append(0xEE)
                    i += 2
                    continue
                else:
                    frames.append(bytes(body))
                    body = bytearray()
                    i += 2
                    continue
            elif b == 0xEE and i + 1 == ln:
                body.append(b)
                i += 1
                continue
            else:
                body.append(b)
                i += 1
    if body:
        frames.append(bytes(body))
    return frames


def xor_at(buf, key, off=0):
    return bytes(b ^ key[(i + off) % 4] for i, b in enumerate(buf))


def score_key_byte0(frames, key4):
    """用 byte0 的合理性给 key0 打分：解出的小值 op 占比。"""
    ok = 0
    for fr in frames:
        if not fr:
            continue
        if (fr[0] ^ key4[0]) < 0x30:
            ok += 1
    return ok / max(1, len(frames))


def main():
    path = sys.argv[1]
    dump_n = 0
    if '--dump' in sys.argv:
        dump_n = int(sys.argv[sys.argv.index('--dump') + 1])

    streams = pcaplib.build_streams(path)
    for st in streams:
        print(f'stream {st.client} -> {st.server}  c2s_pkts={len(st.c2s)} s2c_pkts={len(st.s2c)}')
    # 主游戏流 = S2C 帧数最多的
    st = max(streams, key=lambda s: len(pcaplib.stream_frames(s.s2c)))
    print(f'\n== game stream: {st.client} -> {st.server}')

    s2c_frames = pcaplib.stream_frames(st.s2c)
    print(f'S2C frames={len(s2c_frames)}  op dist: '
          + str(Counter(hex(f[0]) for _, f in s2c_frames).most_common(8)))

    chunks = st.c2s
    frames = raw_frames(chunks)
    print(f'\nC2S raw frames={len(frames)} (pre-XOR)')
    print('len histogram top10:', Counter(len(f) for f in frames).most_common(10))

    # A. 现行方案：帧内 i%4 @0
    opsA = [f[0] ^ KEY[0] for f in frames if f]
    print('\n[A] per-frame phase0 CC3B6DFA: byte0 dist',
          Counter(hex(v) for v in opsA).most_common(12),
          'small(<0x30) ratio=%.2f' % (sum(1 for v in opsA if v < 0x30) / max(1, len(opsA))))

    # B. 单字节暴力：哪个 k0 使 byte0 全落在小值
    best = []
    for k in range(256):
        r = sum(1 for f in frames if f and (f[0] ^ k) < 0x30) / max(1, len(frames))
        best.append((r, k))
    best.sort(reverse=True)
    print('\n[B] byte0 brute k0 top6:', [(f'{k:02X}', f'{r:.2f}') for r, k in best[:6]])

    # C. 4 字节 key 逐位暴力：byte0 用上面 k0，byte1..3 用「解出后 tvarlist/结构合理性」难做，
    #    改用连续相位假设检验：key 相位跨帧连续（未转义体累计）。
    for phase_mode in ('cont_plain', 'cont_wire'):
        out, phase = [], 0
        wire = bytearray()
        for _, payload in chunks:
            wire.extend(payload)
        wi = 0
        body = bytearray()
        frames_ph = []
        i, ln = 0, len(wire)
        while i < ln:
            b = wire[i]
            if b == 0xEE and i + 1 < ln:
                nxt = wire[i + 1]
                if nxt == 0x00:
                    body.append(0xEE)
                    i += 2
                    continue
                else:
                    frames_ph.append((phase, bytes(body)))
                    if phase_mode == 'cont_plain':
                        phase = (phase + len(body)) % 4
                    else:
                        phase = (phase + len(body) + 2) % 4
                    body = bytearray()
                    i += 2
                    continue
            elif b == 0xEE and i + 1 == ln:
                body.append(b)
                i += 1
                continue
            else:
                body.append(b)
                i += 1
        good = sum(1 for ph, fr in frames_ph if fr and (fr[0] ^ KEY[ph]) < 0x30)
        ops = Counter(hex(fr[0] ^ KEY[ph]) for ph, fr in frames_ph if fr)
        print(f'[C:{phase_mode}] continuous phase: small ratio={good / max(1, len(frames_ph)):.2f}',
              ops.most_common(8))

    # D. KEY 旋转族（帧内相位 1/2/3）
    for off in (1, 2, 3):
        good = sum(1 for f in frames if f and (f[0] ^ KEY[off]) < 0x30)
        ops = Counter(hex(f[0] ^ KEY[off]) for f in frames if f)
        print(f'[D] rotate off={off}: small ratio={good / max(1, len(frames)):.2f}',
              ops.most_common(6))

    # E. 用 B 的 k0 逐字节解前 32 帧 dump
    if dump_n:
        k0 = best[0][1]
        print(f'\n[E] dump first {dump_n} frames with k0={k0:02X} (only byte0 keyed, rest raw):')
        for i, f in enumerate(frames[:dump_n]):
            print(f'  [{i:3d}] len={len(f):4d} op?={f[0] ^ k0:02X} raw={f[:32].hex()}')


if __name__ == '__main__':
    main()
