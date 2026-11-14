# -*- coding: utf-8 -*-
"""Enumerate ALL running kernel drivers; flag any whose image is outside
C:\\Windows or whose name looks randomized (Snail SDProtect loads under a
temp/random service key)."""
import ctypes
import ctypes.wintypes as wt
import re

adv = ctypes.WinDLL("advapi32", use_last_error=True)

SERVICE_DRIVER = 0x0B          # SERVICE_KERNEL_DRIVER|SERVICE_FILE_SYSTEM_DRIVER
SERVICE_STATE_ALL = 0x03
SC_ENUM_PROCESS_INFO = 0

SC_MANAGER_ALL_ACCESS = 0xF003F


class ENUM_SERVICE_STATUS_PROCESSW(ctypes.Structure):
    class _S(ctypes.Union):
        class _N(ctypes.Structure):
            _fields_ = [("lpServiceName", wt.LPWSTR), ("lpDisplayName", wt.LPWSTR)]
        _fields_ = [("names", _N), ("lpServiceName2", wt.LPWSTR)]
    _fields_ = [("svc", _S), ("ServiceStatus", wt.BYTE * 36)]  # SERVICE_STATUS_PROCESS


hscm = adv.OpenSCManagerW(None, None, SC_MANAGER_ALL_ACCESS)
if not hscm:
    print("OpenSCManager failed (need admin? listing may still work read-only)")
    hscm = adv.OpenSCManagerW(None, None, 0x0004)  # ENUMERATE_SERVICE
need = wt.DWORD()
count = wt.DWORD()
resume = wt.DWORD()
adv.EnumServicesStatusExW(hscm, SC_ENUM_PROCESS_INFO, SERVICE_DRIVER, SERVICE_STATE_ALL,
                          None, 0, ctypes.byref(need), ctypes.byref(count), ctypes.byref(resume), None)
buf = ctypes.create_string_buffer(need.value + 4096)
ok = adv.EnumServicesStatusExW(hscm, SC_ENUM_PROCESS_INFO, SERVICE_DRIVER, SERVICE_STATE_ALL,
                               buf, need.value + 4096, ctypes.byref(need), ctypes.byref(count),
                               ctypes.byref(resume), None)
print("running driver services: %d (ok=%s)" % (count.value, bool(ok)))

# open each to get ImagePath
rand_re = re.compile(r"^[a-z0-9]{6,12}$", re.I)
hits = 0
items = (ENUM_SERVICE_STATUS_PROCESSW * count.value).from_buffer(buf)
for it in items:
    name = it.svc.names.lpServiceName
    disp = it.svc.names.lpDisplayName or ""
    h = adv.OpenServiceW(hscm, name, 0x0001)  # SERVICE_QUERY_CONFIG
    path = ""
    if h:
        q = adv.QueryServiceConfigW
        q.restype = wt.BOOL
        need2 = wt.DWORD()
        q(h, None, 0, ctypes.byref(need2))
        class QSCW(ctypes.Structure):
            _fields_ = [("dwServiceType", wt.DWORD), ("dwStartType", wt.DWORD),
                        ("dwErrorControl", wt.DWORD), ("lpBinaryPathName", wt.LPWSTR),
                        ("lpLoadOrderGroup", wt.LPWSTR), ("dwTagId", wt.DWORD),
                        ("lpDependencies", wt.LPWSTR), ("lpServiceStartName", wt.LPWSTR),
                        ("lpDisplayName", wt.LPWSTR)]
        cbuf = ctypes.create_string_buffer(need2.value + 4096)
        if q(h, ctypes.cast(cbuf, ctypes.POINTER(QSCW)), need2.value + 4096, ctypes.byref(need2)):
            cfg = ctypes.cast(cbuf, ctypes.POINTER(QSCW)).contents
            path = cfg.lpBinaryPathName or ""
        adv.CloseServiceHandle(h)
    low = path.lower()
    outside = low and "c:\\windows" not in low
    randname = bool(rand_re.match(name)) and "windows" not in low
    if outside or randname:
        hits += 1
        print("  NAME=%-16s PATH=%s" % (name, path))
if not hits:
    print("  (no driver outside C:\\Windows and no random-named driver)")

adv.CloseServiceHandle(hscm)
