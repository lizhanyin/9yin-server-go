# -*- coding: utf-8 -*-
"""Dump the S2C frame window around every 0x28 frame for a given obj id,
plus the C2S 0x07 request positions for the same obj. Usage:
python -I npc_click_window.py <s2c.bin> <c2s.bin> <objid>
"""
import importlib.util, os, struct, sys

SMOKE = r"G:\pojie\ai\9yin-server\tools\smoke"
_spec = importlib.util.spec_from_file_location("capture_deep", os.path.join(SMOKE, "capture_deep.py"))
_cd = importlib.util.module_from_spec(_spec); _spec.loader.exec_module(_cd)

def summarize(fr):
    if not fr:
        return "(empty)"
    op = fr[0]
    if op == 0x28 and len(fr) > 11:
        cnt = struct.unpack_from('<H', fr, 70)[0]
        obj, own = struct.unpack_from('<II', fr, 10)
        return f"0x28 m={fr[1]:#04x} obj={obj} own={own} f={cnt} len={len(fr)}"
    if op == 0x0D and len(fr) >= 0x40:
        obj, own = struct.unpack_from('<II', fr, 1)
        cnt = struct.unpack_from('<H', fr, 0x3E)[0]
        return f"0x0D obj={obj} own={own} f={cnt} len={len(fr)}"
    if op == 0x1E and len(fr) > 8:
        try:
            msg = struct.unpack_from('<i', fr, 5)[0]
        except Exception:
            msg = -1
        return f"0x1E custom msg={msg} len={len(fr)}"
    if op == 0x10 and len(fr) >= 14:
        obj, own = struct.unpack_from('<II', fr, 2)
        idx = struct.unpack_from('<H', fr, 12)[0]
        return f"0x10 prop obj={obj} own={own} idx={idx} len={len(fr)}"
    if op == 0x1F and len(fr) >= 25:
        obj = struct.unpack_from('<I', fr, 1)[0]
        return f"0x1F loc obj={obj} len={len(fr)}"
    if op == 0x31:
        return f"0x31 linkpair len={len(fr)}"
    return f"0x{op:02X} len={len(fr)}"

def main():
    s2c, c2s, target = sys.argv[1], sys.argv[2], int(sys.argv[3])
    sf = _cd.load(s2c)
    hits = []
    for i, fr in enumerate(sf):
        if fr and fr[0] == 0x28 and len(fr) >= 72:
            obj = struct.unpack_from('<I', fr, 10)[0]
            if obj == target:
                hits.append(i)
    print(f"S2C {len(sf)} frames; 0x28 frames for obj {target} at {hits}")
    cf = _cd.load(c2s)
    reqs = []
    for i, fr in enumerate(cf):
        if fr and fr[0] == 0x07 and len(fr) >= 29:
            obj = struct.unpack_from('<I', fr, 21)[0]
            if obj == target:
                reqs.append(i)
    print(f"C2S {len(cf)} frames; 0x07 requests for obj {target} at {reqs}")
    shown = set()
    for h in hits:
        for j in range(max(0, h - 2), min(len(sf), h + 4)):
            if j not in shown:
                shown.add(j)
                mark = " <<<" if j == h else ""
                print(f"[{j:5d}] {summarize(sf[j])}{mark}")
        print("---")

if __name__ == '__main__':
    main()
