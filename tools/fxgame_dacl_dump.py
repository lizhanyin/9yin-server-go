# -*- coding: utf-8 -*-
"""Final elevated dump: full 80-ACE DACL via GetExplicitEntriesFromAcl,
protection-related modules, fxgame in-memory section table + decrypt state."""
import ctypes
import ctypes.wintypes as wt
import datetime
import os
import struct

OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "fxgame_dacl_dump_result.txt")
LINES = []


def log(s):
    print(s)
    LINES.append(s)


k32 = ctypes.WinDLL("kernel32", use_last_error=True)
adv = ctypes.WinDLL("advapi32", use_last_error=True)
psapi = ctypes.WinDLL("psapi", use_last_error=True)
adv.ConvertSidToStringSidW.argtypes = [ctypes.c_void_p, ctypes.POINTER(ctypes.c_wchar_p)]
psapi.GetModuleBaseNameW.argtypes = [wt.HANDLE, ctypes.c_void_p, wt.LPWSTR, wt.DWORD]
psapi.GetModuleFileNameExW.argtypes = [wt.HANDLE, ctypes.c_void_p, wt.LPWSTR, wt.DWORD]
psapi.GetModuleInformation.argtypes = [wt.HANDLE, ctypes.c_void_p, ctypes.c_void_p, wt.DWORD]

PROCESS_QUERY_INFORMATION = 0x0400
PROCESS_VM_READ = 0x0010
READ_CONTROL = 0x00020000

log("== fxgame DACL full dump %s ==" % datetime.datetime.now().isoformat())

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
log("pid=%d" % pid)

h = k32.OpenProcess(PROCESS_QUERY_INFORMATION | PROCESS_VM_READ | READ_CONTROL, False, pid)
if not h:
    log("open failed gle=%d" % ctypes.get_last_error())
    open(OUT, "w", encoding="utf-8").write("\n".join(LINES))
    raise SystemExit(1)


def sid_str(p):
    s = ctypes.c_wchar_p()
    if adv.ConvertSidToStringSidW(p, ctypes.byref(s)):
        return s.value
    return "?"


def acct(p):
    nm = ctypes.create_unicode_buffer(256)
    dom = ctypes.create_unicode_buffer(256)
    use = wt.DWORD()
    if adv.LookupAccountSidW(None, p, nm, ctypes.byref(wt.DWORD(256)), dom,
                             ctypes.byref(wt.DWORD(256)), ctypes.byref(use)):
        return "%s\\%s" % (dom.value, nm.value)
    return "(unresolved)"


# --- 1. DACL via GetExplicitEntriesFromAcl ------------------------------
psd = ctypes.c_void_p()
pdac = ctypes.c_void_p()
st = adv.GetSecurityInfo(h, 6, 0x5, None, None, ctypes.byref(pdac), None, ctypes.byref(psd))
log("GetSecurityInfo(OWNER|DACL) st=%d" % st)


class TRUSTEE_W(ctypes.Structure):
    class _N(ctypes.Union):
        _fields_ = [("ptgName", ctypes.c_void_p), ("pSid", ctypes.c_void_p)]
    _fields_ = [("pMultipleTrustee", ctypes.c_void_p),
                ("MultipleTrusteeOperation", wt.DWORD),
                ("TrusteeForm", wt.DWORD), ("TrusteeType", wt.DWORD), ("name", _N)]


class EXPLICIT_ACCESSW(ctypes.Structure):
    _fields_ = [("grfAccessPermissions", wt.DWORD), ("grfAccessMode", wt.DWORD),
                ("grfInheritance", wt.DWORD), ("Trustee", TRUSTEE_W)]


NAMES = {0x1: "TERMINATE", 0x2: "CREATE_THREAD", 0x4: "SET_SESSIONID", 0x8: "VM_OPERATION",
         0x10: "VM_READ", 0x20: "VM_WRITE", 0x40: "DUP_HANDLE", 0x80: "CREATE_PROCESS",
         0x100: "SUSPEND_RESUME", 0x200: "READ_CONTROL", 0x400: "QUERY_INFO", 0x800: "SET_INFO",
         0x1000: "QUERY_LIMITED", 0x10000: "SYNCHRONIZE"}


def accstr(m):
    if m in (0x1F0FFF, 0x1FFFFF):
        return "ALL_ACCESS"
    if m in (0x80000000, 0x40000000, 0x20000000, 0x10000000):
        return {0x80000000: "GENERIC_READ", 0x40000000: "GENERIC_WRITE",
                0x20000000: "GENERIC_EXECUTE", 0x10000000: "GENERIC_ALL"}[m]
    return "|".join(n for b, n in NAMES.items() if m & b) or hex(m)


if st == 0 and pdac.value:
    pe = ctypes.POINTER(EXPLICIT_ACCESSW)()
    cnt = wt.ULONG()
    if adv.GetExplicitEntriesFromAclW(pdac, ctypes.byref(cnt), ctypes.byref(pe)):
        log("DACL ACEs: %d" % cnt.value)
        # TRUSTEE form: multiple strings form used by GetExplicitEntries - trustee.p multiple strings ptr
        # In EXPLICIT_ACCESS from GetExplicitEntriesFromAcl, TrusteeName points to SID (ptgName)
        items = ctypes.cast(pe, ctypes.POINTER(EXPLICIT_ACCESSW))
        for i in range(cnt.value):
            ea = items[i]
            sidp = ea.Trustee.name.pSid  # TRUSTEE_IS_SID -> points to SID
            who = acct(ctypes.c_void_p(sidp or 0)) if sidp else "?"
            s = sid_str(ctypes.c_void_p(sidp)) if sidp else "?"
            mode = {1: "GRANT", 2: "SET", 3: "DENY", 4: "REVOKE"}.get(ea.grfAccessMode, ea.grfAccessMode)
            log("  [%02d] %-6s %-40s %-46s %s" % (i, mode, who, s, accstr(ea.grfAccessPermissions)))
        adv.LocalFree(ctypes.c_void_p(ctypes.addressof(items.contents)))
    else:
        log("GetExplicitEntriesFromAcl failed gle=%d" % ctypes.get_last_error())

# --- 2. modules + section table ----------------------------------------
arr = (wt.HMODULE * 2048)()
need = wt.DWORD()
base = None
prot_mods = []
if psapi.EnumProcessModulesEx(h, arr, ctypes.sizeof(arr), ctypes.byref(need), 3):
    cnt2 = need.value // ctypes.sizeof(wt.HMODULE)
    log("\nmodules: %d" % cnt2)
    for i in range(cnt2):
        nm = ctypes.create_unicode_buffer(260)
        psapi.GetModuleBaseNameW(h, arr[i], nm, 260)
        fn = ctypes.create_unicode_buffer(1024)
        psapi.GetModuleFileNameExW(h, arr[i], fn, 1024)

        class MI(ctypes.Structure):
            _fields_ = [("lpBaseOfDll", ctypes.c_void_p), ("SizeOfImage", wt.DWORD), ("EntryPoint", ctypes.c_void_p)]
        mi = MI()
        psapi.GetModuleInformation(h, arr[i], ctypes.byref(mi), ctypes.sizeof(mi))
        if nm.value.lower() == "fxgame.exe":
            base = mi.lpBaseOfDll or 0
        low = nm.value.lower()
        if any(k2 in low for k2 in (".sn", "spdc", "protect", "vmp", "snail", "sd")):
            prot_mods.append("%-16s base=0x%X %s" % (nm.value, mi.lpBaseOfDll or 0, fn.value))
    log("protection-related modules:")
    for m in prot_mods:
        log("  " + m)
else:
    log("EnumProcessModulesEx failed gle=%d" % ctypes.get_last_error())

if base:
    log("\nfxgame base=0x%X" % base)
    buf = ctypes.create_string_buffer(0x4000)
    got = ctypes.c_size_t()
    if k32.ReadProcessMemory(h, ctypes.c_void_p(base), buf, 0x4000, ctypes.byref(got)):
        hdr = buf.raw[:got.value]
        e_lfanew = struct.unpack_from("<I", hdr, 0x3C)[0]
        sig = hdr[e_lfanew:e_lfanew + 4]
        log("PE header read OK sig=%s" % repr(sig))
        if sig == b"PE\0\0":
            nsec, optsz = struct.unpack_from("<HH", hdr, e_lfanew + 6)
            ep = struct.unpack_from("<I", hdr, e_lfanew + 24 + 16)[0]
            soff = e_lfanew + 24 + optsz
            log("sections=%d EP_RVA=0x%X" % (nsec, ep))
            secs = []
            for i in range(nsec):
                s = hdr[soff + 40 * i: soff + 40 * (i + 1)]
                if len(s) < 40:
                    log("  (section table truncated at %d, e_lfanew=0x%X past read window)" % (i, e_lfanew))
                    break
                nm = s[:8].rstrip(b"\0").decode("latin1", "replace")
                vsz, va = struct.unpack_from("<II", s, 8)
                log("  %-8s VA=0x%06X VSZ=0x%-6X%s" % (nm, va, vsz,
                    " <== EP" if va <= ep < va + max(vsz, 1) else ""))
                secs.append((nm, va))
            for nm, va in secs:
                b2 = ctypes.create_string_buffer(4096)
                g2 = ctypes.c_size_t()
                if k32.ReadProcessMemory(h, ctypes.c_void_p(base + va + 0x1000), b2, 4096, ctypes.byref(g2)):
                    nz = sum(1 for x in b2.raw[:g2.value] if x)
                    log("  %-8s runtime page nonzero=%d/4096" % (nm, nz))

log("\nDONE")
open(OUT, "w", encoding="utf-8").write("\n".join(LINES))
