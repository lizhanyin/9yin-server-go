# -*- coding: utf-8 -*-
"""Determine WHY fxgame.exe denies VM_READ: DACL vs PPL vs kernel callback."""
import ctypes
import ctypes.wintypes as wt

k32 = ctypes.WinDLL("kernel32", use_last_error=True)
ntdll = ctypes.WinDLL("ntdll", use_last_error=True)

PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
PROCESS_VM_READ = 0x0010

PID = 58840
CTRL = None  # control pid filled below


class PS_PROTECTION(ctypes.Structure):
    _fields_ = [("Level", ctypes.c_ubyte)]


def get_path(pid):
    h = k32.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, False, pid)
    if not h:
        return None
    buf = ctypes.create_unicode_buffer(1024)
    sz = wt.DWORD(1024)
    ok = k32.QueryFullProcessImageNameW(h, 0, buf, ctypes.byref(sz))
    k32.CloseHandle(h)
    return buf.value if ok else None


# find own python pid + a control process (explorer) for comparison
k32.GetCurrentProcessId.restype = wt.DWORD
ctrl_candidates = []
import ctypes.wintypes as wt2
TH32CS_SNAPPROCESS = 2


class PE32W(ctypes.Structure):
    _fields_ = [("dwSize", wt.DWORD), ("cntUsage", wt.DWORD), ("th32ProcessID", wt.DWORD),
                ("th32DefaultHeapID", ctypes.POINTER(ctypes.c_ulong)), ("th32ModuleID", wt.DWORD),
                ("cntThreads", wt.DWORD), ("th32ParentProcessID", wt.DWORD),
                ("pcPriClassBase", ctypes.c_long), ("dwFlags", wt.DWORD),
                ("szExeFile", wt.WCHAR * 260)]


snap = k32.CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0)
e = PE32W()
e.dwSize = ctypes.sizeof(e)
if k32.Process32FirstW(snap, ctypes.byref(e)):
    while True:
        if e.th32ProcessID != PID:
            if e.szExeFile.lower() in ("explorer.exe",):
                ctrl_candidates.append(e.th32ProcessID)
        if not k32.Process32NextW(snap, ctypes.byref(e)):
            break
k32.CloseHandle(snap)

print("fxgame image path :", get_path(PID))
for c in ctrl_candidates[:1]:
    print("control explorer  : pid=%d %s" % (c, get_path(c)))

# --- control test: VM_READ on explorer (same user, unprotected) --------
for c in ctrl_candidates[:1]:
    hh = k32.OpenProcess(PROCESS_VM_READ, False, c)
    print("control OpenProcess(VM_READ explorer) -> %s" % ("OK" if hh else "DENIED gle=%d" % ctypes.get_last_error()))
    if hh:
        k32.CloseHandle(hh)

# --- PPL check (works with LIMITED handle, info class 61) --------------
h = k32.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, False, PID)
pp = PS_PROTECTION()
st = ntdll.NtQueryInformationProcess(h, 61, ctypes.byref(pp), ctypes.sizeof(pp), None)
lvl = pp.Level
print("\nProcessProtectionInformation status=0x%08X Level=0x%02X" % (st & 0xFFFFFFFF, lvl))
if st == 0:
    ptype = lvl & 0x07
    pl = (lvl >> 4) & 0x07
    tnames = {0: "None(Plain)", 1: "Light(PPL)", 2: "Light+Authenticode", 3: "Protected"}
    print("  -> type=%s signer=%d %s" % (tnames.get(ptype, hex(ptype)), pl,
          "(if type!=0, denial comes from PPL)" if ptype else "(plain process -> denial is DACL or driver callback)"))

# --- signature check: info class 99? not stable; check kernel drivers --
import subprocess
print("\n== loaded kernel drivers matching protect/snail/sd/anti ==")
try:
    out = subprocess.check_output(["driverquery", "/FO", "CSV"], stderr=subprocess.STDOUT).decode("gbk", "replace")
    for line in out.splitlines():
        low = line.lower()
        if any(k in low for k in ("protect", "snail", "sdprotect", "sprotect", "spdc", "anti", "guard", "tp", "sina")):
            print("  " + line.strip())
except Exception as ex:
    print("  driverquery failed: %s" % ex)
