# -*- coding: utf-8 -*-
"""Registry-based driver scan: all kernel/fs driver services with ImagePath
outside C:\\Windows, then check running state via SCM (unprivileged path)."""
import winreg
import re
import subprocess

roots = [
    (winreg.HKEY_LOCAL_MACHINE, r"SYSTEM\CurrentControlSet\Services"),
]

suspects = []
rand_re = re.compile(r"^[A-Za-z0-9]{6,12}$")
for hive, path in roots:
    k = winreg.OpenKey(hive, path)
    i = 0
    while True:
        try:
            sub = winreg.EnumKey(k, i)
        except OSError:
            break
        i += 1
        try:
            sk = winreg.OpenKey(hive, path + "\\" + sub)
            try:
                typ = winreg.QueryValueEx(sk, "Type")[0]
            except OSError:
                continue
            if typ not in (1, 2):  # kernel driver / fs driver
                continue
            try:
                img = winreg.QueryValueEx(sk, "ImagePath")[0]
            except OSError:
                img = ""
            low = img.lower()
            outside = low and "c:\\windows" not in low and "systemroot" not in low
            randname = bool(rand_re.match(sub))
            if outside or randname:
                suspects.append((sub, typ, img, randname, outside))
        except OSError:
            continue

print("candidate driver services (non-system path or random name): %d" % len(suspects))
for name, typ, img, rnd, out in suspects:
    state = "?"
    try:
        o = subprocess.check_output(["sc", "query", name], stderr=subprocess.STDOUT).decode("gbk", "replace")
        for line in o.splitlines():
            if "STATE" in line or "状态" in line:
                state = line.split(":", 1)[1].strip()[:30]
                break
    except Exception:
        pass
    print("  %-20s type=%d state=%-32s img=%s %s" % (name, typ, state, img, "(RANDOMNAME)" if rnd else ""))
