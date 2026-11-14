# -*- coding: utf-8 -*-
"""Elevated verdict probe: does SeDebugPrivilege bypass fxgame's memory denial?

Runs elevated (UAC). SeDebugPrivilege short-circuits DACL at access check, so:
  read OK    -> protection was user-mode DACL rewrite (SDProtect userland)
  still denied -> kernel ObCallback is stripping rights (hidden driver)

Writes verdict to fxgame_elevated_verdict_result.txt next to this script.
ASCII-only output to avoid console codepage issues."""
import ctypes
import ctypes.wintypes as wt
import datetime
import os
import struct

OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "fxgame_elevated_verdict_result.txt")
LINES = []


def log(s):
    print(s)
    LINES.append(s)


k32 = ctypes.WinDLL("kernel32", use_last_error=True)
adv = ctypes.WinDLL("advapi32", use_last_error=True)
psapi = ctypes.WinDLL("psapi", use_last_error=True)

PROCESS_QUERY_INFORMATION = 0x0400
PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
PROCESS_VM_READ = 0x0010
PROCESS_VM_WRITE = 0x0020
PROCESS_VM_OPERATION = 0x0008
PROCESS_ALL_ACCESS = 0x1FFFFF

log("== elevated verdict probe %s ==" % datetime.datetime.now().isoformat())
log("IsUserAnAdmin=%d" % ctypes.windll.shell32.IsUserAnAdmin())

# --- 1. enable SeDebugPrivilege ---------------------------------------
TOKEN_ADJUST_PRIVILEGES = 0x20
TOKEN_QUERY = 0x8
SE_PRIVILEGE_ENABLED = 0x2


class LUID(ctypes.Structure):
    _fields_ = [("LowPart", wt.DWORD), ("HighPart", ctypes.c_long)]


class TOKEN_PRIVS(ctypes.Structure):
    _fields_ = [("PrivilegeCount", wt.DWORD), ("Luid", LUID), ("Attributes", wt.DWORD)]


hTok = wt.HANDLE()
adv.OpenProcessToken(k32.GetCurrentProcess(), TOKEN_ADJUST_PRIVILEGES | TOKEN_QUERY, ctypes.byref(hTok))
luid = LUID()
ok1 = adv.LookupPrivilegeValueW(None, "SeDebugPrivilege", ctypes.byref(luid))
tp = TOKEN_PRIVS(1, luid, SE_PRIVILEGE_ENABLED)
ok2 = adv.AdjustTokenPrivileges(hTok, False, ctypes.byref(tp), 0, None, None)
gle = k32.GetLastError()
log("SeDebugPrivilege: lookup=%s adjust=%s gle=%d (1300=NOT_ALL_ASSIGNED, 0=enabled)" % (bool(ok1), bool(ok2), gle))
if gle == 1300:
    log("FATAL: SeDebugPrivilege not held - verdict impossible, abort")
    open(OUT, "w", encoding="utf-8").write("\n".join(LINES))
    raise SystemExit(1)

# --- 2. find fxgame pid ------------------------------------------------
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
pid = None
if k32.Process32FirstW(snap, ctypes.byref(e)):
    while True:
        if e.szExeFile.lower() == "fxgame.exe":
            pid = e.th32ProcessID
            break
        if not k32.Process32NextW(snap, ctypes.byref(e)):
            break
k32.CloseHandle(snap)
if not pid:
    log("fxgame.exe not running - abort")
    open(OUT, "w", encoding="utf-8").write("\n".join(LINES))
    raise SystemExit(1)
log("target pid=%d" % pid)

# --- 3. open with SeDebugPrivilege active ------------------------------
h = None
for label, acc in [("QUERY_LIMITED", PROCESS_QUERY_LIMITED_INFORMATION),
                   ("QUERY_INFO", PROCESS_QUERY_INFORMATION),
                   ("VM_READ", PROCESS_VM_READ),
                   ("VM_READ|VM_WRITE|VM_OPERATION", PROCESS_VM_READ | PROCESS_VM_WRITE | PROCESS_VM_OPERATION),
                   ("PROCESS_ALL_ACCESS", PROCESS_ALL_ACCESS)]:
    hh = k32.OpenProcess(acc, False, pid)
    err = ctypes.get_last_error()
    okk = bool(hh)
    log("OpenProcess(%s) -> %s gle=%d" % (label, "OK" if okk else "DENIED", 0 if okk else err))
    if okk and h is None and acc == PROCESS_VM_READ:
        h = hh
    elif okk and acc == PROCESS_ALL_ACCESS:
        k32.CloseHandle(hh)

verdict = None
if h:
    # --- 4. actually read memory ---------------------------------------
    # module base via EnumProcessModulesEx (QUERY_INFO+VM_READ)
    arr = (wt.HMODULE * 2048)()
    need = wt.DWORD()
    base = None
    if psapi.EnumProcessModulesEx(h, arr, ctypes.sizeof(arr), ctypes.byref(need), 3):
        cnt = need.value // ctypes.sizeof(wt.HMODULE)
        log("modules enumerated: %d" % cnt)
        for i in range(cnt):
            nm = ctypes.create_unicode_buffer(260)
            psapi.GetModuleBaseNameW(h, arr[i], nm, 260)
            if nm.value.lower() == "fxgame.exe":
                class MI(ctypes.Structure):
                    _fields_ = [("lpBaseOfDll", ctypes.c_void_p), ("SizeOfImage", wt.DWORD), ("EntryPoint", ctypes.c_void_p)]
                mi = MI()
                psapi.GetModuleInformation(h, arr[i], ctypes.byref(mi), ctypes.sizeof(mi))
                base = mi.lpBaseOfDll or 0
                log("fxgame module base=0x%X" % base)
                break
    else:
        log("EnumProcessModulesEx failed gle=%d" % ctypes.get_last_error())

    if base:
        buf = ctypes.create_string_buffer(0x1000)
        got = ctypes.c_size_t()
        if k32.ReadProcessMemory(h, ctypes.c_void_p(base), buf, 0x1000, ctypes.byref(got)):
            hdr = buf.raw[:got.value]
            e_lfanew = struct.unpack_from("<I", hdr, 0x3C)[0]
            sig = hdr[e_lfanew:e_lfanew + 4]
            log("ReadProcessMemory(PE header) OK %d bytes, PE sig=%s" % (got.value, repr(sig)))
            if sig == b"PE\0\0":
                nsec, optsz = struct.unpack_from("<HH", hdr, e_lfanew + 6)
                ep = struct.unpack_from("<I", hdr, e_lfanew + 24 + 16)[0]
                soff = e_lfanew + 24 + optsz
                log("in-memory sections (%d) EP_RVA=0x%X:" % (nsec, ep))
                for i in range(nsec):
                    s = hdr[soff + 40 * i: soff + 40 * (i + 1)]
                    nm = s[:8].rstrip(b"\0").decode("latin1", "replace")
                    vsz, va = struct.unpack_from("<II", s, 8)
                    log("  %-8s VA=0x%06X VSZ=0x%X%s" % (nm, va, vsz,
                        " <== EP here" if va <= ep < va + max(vsz, 1) else ""))
            verdict = "READ_OK_WITH_SEDEBUG -> protection = user-mode DACL rewrite (no kernel strip)"
        else:
            log("ReadProcessMemory failed gle=%d" % ctypes.get_last_error())
            verdict = "OPEN_OK_BUT_READ_DENIED -> partial (unusual)"
    else:
        verdict = "VM_READ_HANDLE_OK -> DACL bypassed (module enum/read skipped)"
else:
    verdict = "STILL_DENIED_WITH_SEDEBUG -> kernel ObCallback stripping (hidden driver)"

log("VERDICT: " + verdict)
open(OUT, "w", encoding="utf-8").write("\n".join(LINES))
