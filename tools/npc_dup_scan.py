# -*- coding: utf-8 -*-
"""Scan a self-dump s2c.bin for object-creation frames (0x0D / 0x28 family):
list (frame index, op, mode, obj, own, field count) and flag duplicate
obj ids sent more than once. Usage: python -I npc_dup_scan.py <s2c.bin>
"""
import importlib.util, os, struct, sys

SMOKE = r"G:\pojie\ai\9yin-server\tools\smoke"
_spec = importlib.util.spec_from_file_location("capture_deep", os.path.join(SMOKE, "capture_deep.py"))
_cd = importlib.util.module_from_spec(_spec); _spec.loader.exec_module(_cd)

def main():
    path = sys.argv[1]
    frames = _cd.load(path)
    print(f"{os.path.basename(path)}: {len(frames)} frames")
    seen = {}  # objkey -> [(idx, op, cnt)]
    rows = []
    for i, fr in enumerate(frames):
        if not fr:
            continue
        op = fr[0]
        if op == 0x0D:
            if len(fr) < 0x40:
                continue
            obj, own = struct.unpack_from('<II', fr, 1)
            cnt = struct.unpack_from('<H', fr, 0x3E)[0]
            rows.append((i, 0x0D, None, obj, own, cnt))
            seen.setdefault((obj, own), []).append((i, 0x0D, cnt))
        elif op in (0x27, 0x28, 0x29, 0x2A, 0x2D, 0x2E):
            mode = fr[1] if len(fr) > 1 else None
            if op == 0x28 and mode in (0x46, 0x47) and len(fr) >= 0x44:
                body_off = 10
                obj, own = struct.unpack_from('<II', fr, body_off)
                cnt = struct.unpack_from('<H', fr, body_off + 60)[0]
                rows.append((i, 0x28, mode, obj, own, cnt))
                seen.setdefault((obj, own), []).append((i, 0x28, cnt))
    for i, op, mode, obj, own, cnt in rows:
        n = len(seen[(obj, own)])
        flag = "  <== MULTI" if n > 1 else ""
        m = f" mode=0x{mode:02X}" if mode else ""
        print(f"[{i:5d}] op=0x{op:02X}{m} obj={obj}({obj:#x}) own={own} fields={cnt} sends={n}{flag}")
    dups = {k: v for k, v in seen.items() if len(v) > 1}
    print(f"\ntotal creation frames={len(rows)} unique objs={len(seen)} multi-send objs={len(dups)}")
    for k, v in list(dups.items())[:20]:
        print(f"  obj={k[0]} own={k[1]}: {v}")

if __name__ == '__main__':
    main()
