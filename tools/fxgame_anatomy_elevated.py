# -*- coding: utf-8 -*-
"""Elevated anatomy: dump fxgame's rewritten DACL ACEs, token integrity,
in-memory section table, and .text decrypt state. Result -> txt next to script."""
import ctypes
import ctypes.wintypes as wt
import datetime
import os
import struct

OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "fxgame_anatomy_result.txt")
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
psapi.EnumProcessModulesEx.argtypes = [wt.HANDLE, ctypes.c_void_p, wt.DWORD, ctypes.POINTER(wt.DWORD), wt.DWORD]

PROCESS_QUERY_INFORMATION = 0x0400
PROCESS_VM_READ = 0x0010
READ_CONTROL = 0x00020000
TOKEN_QUERY = 0x8
TOKEN_DUPLICATE = 0x2

log("== fxgame anatomy (elevated) %s ==" % datetime.datetime.now().isoformat())

# --- find pid ----------------------------------------------------------
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

acc = PROCESS_QUERY_INFORMATION | PROCESS_VM_READ | READ_CONTROL
h = k32.OpenProcess(acc, False, pid)
if not h:
    log("open failed gle=%d" % ctypes.get_last_error())
    open(OUT, "w", encoding="utf-8").write("\n".join(LINES))
    raise SystemExit(1)
log("handle OK (QUERY_INFO|VM_READ|READ_CONTROL)")

# --- 1. DACL anatomy ----------------------------------------------------
psid_owner = ctypes.c_void_p()
pdac = ctypes.c_void_p()
psacl = ctypes.c_void_p()
psd = ctypes.c_void_p()
st = adv.GetSecurityInfo(h, 6, 0x5 | 0x1, ctypes.byref(psid_owner), None, ctypes.byref(pdac),
                         ctypes.byref(psacl), ctypes.byref(psd))  # OWNER|GROUP|DACL|SACL
log("\n-- security descriptor (SACL needs SE_SECURITY_NAME; may fail) st=%d --" % st)


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


if st == 0:
    log("owner: %s %s" % (acct(psid_owner), sid_str(psid_owner)))
    if pdac.value:
        class ACL(ctypes.Structure):
            _fields_ = [("AclRevision", ctypes.c_ubyte), ("Sbz1", ctypes.c_ubyte),
                        ("AclCount", wt.WORD), ("Sbz2", wt.WORD)]
        a = ctypes.cast(pdac, ctypes.POINTER(ACL)).contents
        log("DACL present: revision=%d ACECount=%d" % (a.AclRevision, a.AclCount))
        PROCESS_VM_READ_ = 0x10
        NAMES = {0x1: "TERMINATE", 0x8: "VM_OPERATION", 0x10: "VM_READ", 0x20: "VM_WRITE",
                 0x40: "DUP_HANDLE", 0x200: "READ_CONTROL", 0x400: "QUERY_INFO",
                 0x1000: "QUERY_LIMITED", 0x800: "SET_INFO", 0x100: "SUSPEND_RESUME",
                 0x2: "CREATE_THREAD", 0x4: "SET_SESSIONID", 0x10000: "SYNCHRONIZE"}
        GENERIC = {0x80000000: "GENERIC_READ", 0x40000000: "GENERIC_WRITE",
                   0x20000000: "GENERIC_EXECUTE", 0x10000000: "GENERIC_ALL"}

        def accstr(m):
            if m in GENERIC:
                return GENERIC[m]
            if m == 0x1F0FFF or m == 0x1FFFFF:
                return "ALL_ACCESS"
            return "|".join(n for b, n in NAMES.items() if m & b) or hex(m)

        # walk ACEs: header 4 bytes (type,flags,size)
        p = pdac.value + 8  # skip ACL header
        for i in range(a.AclCount):
            hdr = ctypes.cast(p, ctypes.POINTER(ctypes.c_ubyte * 4)).contents
            atype, flags, size = hdr[0], hdr[1], hdr[2] | (hdr[3] << 8)
            mask = struct.unpack_from("<I", ctypes.string_at(p + 4, size - 4), 0)[0]
            sid = p + 8
            who = acct(ctypes.c_void_p(sid))
            tnames = {0: "ALLOW", 1: "DENY", 2: "AUDIT", 3: "ALARM", 4: "ALLOW-COMPOUND",
                      5: "ALLOW-OBJECT", 6: "DENY-OBJECT"}
            log("  ACE%d %s %s mask=0x%08X (%s)" % (i, tnames.get(atype, atype), who, mask, accstr(mask)))
            p += size
    else:
        log("DACL = NULL (everyone full - not our case)")
else:
    log("GetSecurityInfo failed gle=%d" % ctypes.get_last_error())

# --- 2. target token: user + integrity ---------------------------------
hTokT = wt.HANDLE()
if adv.OpenProcessToken(h, TOKEN_QUERY, ctypes.byref(hTokT)):
    # TokenUser=1, TokenIntegrityLevel=25, TokenElevationType=18
    buf = ctypes.create_string_buffer(256)
    ret = wt.DWORD()
    if adv.GetTokenInformation(hTokT, 1, buf, 256, ctypes.byref(ret)):
        pu = ctypes.cast(buf, ctypes.POINTER(ctypes.c_void_p)).contents.value
        log("\ntarget runs as: %s %s" % (acct(ctypes.c_void_p(pu)), sid_str(ctypes.c_void_p(pu))))
    if adv.GetTokenInformation(hTokT, 25, buf, 256, ctypes.byref(ret)):
        # TOKEN_MANDATORY_LABEL { Sid, Attributes }
        psid = struct.unpack_from("<Q", buf.raw, 0)[0]
        s = sid_str(ctypes.c_void_p(psid))
        # integrity = last subauthority
        lvl = int(s.rstrip(">").split("-")[-1])
        iname = {0x2000: "Low", 0x3000: "Medium", 0x4000: "High", 0x5000: "System"}.get(lvl, hex(lvl))
        log("target integrity: %s (%s)" % (iname, s))
    k32.CloseHandle(hTokT)

# --- 3. modules + in-memory PE ------------------------------------------
arr = (wt.HMODULE * 2048)()
need = wt.DWORD()
base = None
if psapi.EnumProcessModulesEx(h, arr, ctypes.sizeof(arr), ctypes.byref(need), 3):
    cnt = need.value // ctypes.sizeof(wt.HMODULE)
    log("\nmodules: %d" % cnt)
    prot_mods = []
    for i in range(cnt):
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
        if any(k in low for k in (".sn", "spdc", "protect", "vmp", "sina")):
            prot_mods.append("%s -> %s (base=0x%X)" % (nm.value, fn.value, mi.lpBaseOfDll or 0))
    log("protection-related modules:")
    for m in prot_mods:
        log("  " + m)
else:
    log("EnumProcessModulesEx failed gle=%d" % ctypes.get_last_error())

if base:
    log("fxgame base=0x%X" % base)
    buf = ctypes.create_string_buffer(0x2000)
    got = ctypes.c_size_t()
    if k32.ReadProcessMemory(h, ctypes.c_void_p(base), buf, 0x2000, ctypes.byref(got)):
        hdr = buf.raw[:got.value]
        e_lfanew = struct.unpack_from("<I", hdr, 0x3C)[0]
        sig = hdr[e_lfanew:e_lfanew + 4]
        log("PE header read OK, sig=%s" % repr(sig))
        if sig == b"PE\0\0":
            nsec, optsz = struct.unpack_from("<HH", hdr, e_lfanew + 6)
            ep = struct.unpack_from("<I", hdr, e_lfanew + 24 + 16)[0]
            soff = e_lfanew + 24 + optsz
            log("sections=%d EP_RVA=0x%X" % (nsec, ep))
            for i in range(nsec):
                s = hdr[soff + 40 * i: soff + 40 * (i + 1)]
                nm = s[:8].rstrip(b"\0").decode("latin1", "replace")
                vsz, va = struct.unpack_from("<II", s, 8)
                log("  %-8s VA=0x%06X VSZ=0x%-6X%s" % (nm, va, vsz,
                    " <== EP" if va <= ep < va + max(vsz, 1) else ""))
            # .text decrypt check: read first page of each exec section
            for i in range(nsec):
                s = hdr[soff + 40 * i: soff + 40 * (i + 1)]
                nm = s[:8].rstrip(b"\0").decode("latin1", "replace")
                va, = struct.unpack_from("<I", s, 12)
                b2 = ctypes.create_string_buffer(4096)
                g2 = ctypes.c_size_t()
                if k32.ReadProcessMemory(h, ctypes.c_void_p(base + va + 0x100), b2, 4096, ctypes.byref(g2)):
                    nz = sum(1 for x in b2.raw[:g2.value] if x)
                    log("  %-8s runtime first-4K nonzero=%d/4096" % (nm, nz))

log("\nDONE")


def main():
    pass


try:
    main()
except Exception:
    import traceback
    log("EXCEPTION:\n" + traceback.format_exc())
open(OUT, "w", encoding="utf-8").write("\n".join(LINES))
