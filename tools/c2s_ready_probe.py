# -*- coding: utf-8 -*-
"""验证「C2S 单字节 0x09 = 场景就绪信号 → 触发第二爆发+AOI 物化」假说。

对每个抓包的游戏流：
  - 列出全部 C2S 非 0x0A/0x18 帧（控制面）
  - 对每个 C2S 0x09：其后 150ms 内 S2C 帧数/op 分布、之后首个 0x28/0x0D 的延迟
  - 0x0B 到首个 C2S 0x09 的间隔（=客户端读条时长）

Usage: python -I c2s_ready_probe.py <cap.pcapng> [...]
"""
import sys
from collections import Counter

sys.path.insert(0, r'G:\pojie\ai\9yin-server-go-2433\tools\pcap_analysis')
import pcaplib  # noqa: E402

sys.stdout.reconfigure(encoding='utf-8')


def game_streams(path):
    out = []
    for st in pcaplib.build_streams(path):
        s2c = [(t, f) for t, f in pcaplib.stream_frames(st.s2c) if f]
        if len(s2c) < 200:
            continue
        c2s = [(t, f) for t, f in pcaplib.stream_frames(st.c2s) if f]
        out.append((st, c2s, s2c))
    return out


def main():
    for path in sys.argv[1:]:
        print('=' * 100)
        print(path)
        for st, c2s, s2c in game_streams(path):
            t0 = min(c2s[0][0], s2c[0][0])
            print(f'-- stream {st.client} -> {st.server}  C2S={len(c2s)} S2C={len(s2c)}')
            print('   C2S op dist:', Counter('%02X' % f[0] for _, f in c2s).most_common())
            ctl = [(t, f) for t, f in c2s if f[0] not in (0x0A, 0x18)]
            for t, f in ctl[:12]:
                print('   C2S ctl @%.3fs op%02X len=%d %s' % ((t - t0) / 1e6, f[0], len(f), f[:16].hex()))
            scene = [(t, f) for t, f in s2c if f[0] == 0x0B]
            for t, f in scene:
                print('   S2C 0x0B @%.3fs len=%d' % ((t - t0) / 1e6, len(f)))
            ent = [(t, f) for t, f in s2c if f[0] in (0x28, 0x0D)]
            print('   S2C 0x28/0x0D total=%d  times(s): %s' % (
                len(ent), ' '.join('%.3f' % ((t - t0) / 1e6) for t, _ in ent[:14])))
            for t9, f9 in [(t, f) for t, f in c2s if f[0] == 0x09]:
                win = [(t, f) for t, f in s2c if t9 <= t <= t9 + 150_000]
                pre = [(t, f) for t, f in s2c if t9 - 1_000_000 <= t < t9]
                nxt = [(t, f) for t, f in ent if t > t9]
                prev_c = [(t, f) for t, f in c2s if t < t9][-3:]
                print('   >> C2S 0x09 @%.3fs len=%d | S2C 前1s=%d 帧 | 后150ms=%d 帧 %s' % (
                    (t9 - t0) / 1e6, len(f9), len(pre), len(win),
                    Counter('%02X' % f[0] for _, f in win).most_common(6)))
                if win:
                    print('      首帧延迟 %.1fms' % ((win[0][0] - t9) / 1e3))
                if nxt:
                    print('      其后首个 0x28/0x0D 延迟 %.3fs (op%02X len=%d)' % (
                        (nxt[0][0] - t9) / 1e6, nxt[0][1][0], len(nxt[0][1])))
                ent_before = [1 for t, f in ent if t < t9]
                print('      此前已收 0x28/0x0D = %d 帧' % len(ent_before))


if __name__ == '__main__':
    main()
