# -*- coding: utf-8 -*-
"""Probe live fxgame.exe for memory-protection posture (read-only, no writes).

Checks:
 1. OpenProcess with escalating access rights (query / vm_read / vm_write / all)
 2. Full module list (spot packer/anti-cheat DLLs like sprotect.sn / spdc)
 3. fxgame module in-memory PE section table + per-section page protection
 4. Readability of .text (VMProtect decrypt-at-runtime check)
 5. Entry-point section (VMP hallmark: EP in a late appended section)
"""
import ctypes
import ctypes.wintypes as wt
import struct
import sys

k32 = ctypes.WinDLL("kernel32", use_last_error=True)
psapi = ctypes.WinDLL("psapi", use_last_error=True)
ntdll = ctypes.WinDLL("ntdll", use_last_error=True)

PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
PROCESS_QUERY_INFORMATION = 0x0400
PROCESS_VM_READ = 0x0010
PROCESS_VM_WRITE = 0x0020
PROCESS_VM_OPERATION = 0x0008
PROCESS_DUP_HANDLE = 0x0040
PROCESS_ALL_ACCESS = 0x1FFFFF

PAGE_FLAGS = {
    0x01: "NOACCESS", 0x02: "READONLY", 0x04: "READWRITE", 0x08: "WRITECOPY",
    0x10: "EXECUTE", 0x20: "EXECUTE_READ", 0x30: "EXECUTE_READWRITE", 0x40: "EXECUTE_WRITECOPY",
    0x104: "READONLY+GUARD", 0x204: "READWRITE+GUARD",
}


def page_str(p):
    g = "+GUARD" if p & 0x100 else ""
    base = PAGE_FLAGS.get(p & 0xFF, hex(p & 0xFF))
    return base + g


# --- find pid ----------------------------------------------------------
TH32CS_SNAPPROCESS = 0x2


class PROCESSENTRY32W(ctypes.Structure):
    _fields_ = [("dwSize", wt.DWORD), ("cntUsage", wt.DWORD), ("th32ProcessID", wt.DWORD),
                ("th32DefaultHeapID", ctypes.POINTER(ctypes.c_ulong)), ("th32ModuleID", wt.DWORD),
                ("cntThreads", wt.DWORD), ("th32ParentProcessID", wt.DWORD),
                ("pcPriClassBase", ctypes.c_long), ("dwFlags", wt.DWORD),
                ("szExeFile", wt.WCHAR * 260)]


def find_pid(name):
    snap = k32.CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0)
    e = PROCESSENTRY32W()
    e.dwSize = ctypes.sizeof(e)
    pids = []
    if k32.Process32FirstW(snap, ctypes.byref(e)):
        while True:
            if e.szExeFile.lower() == name:
                pids.append((e.th32ProcessID, e.th32ParentProcessID))
            if not k32.Process32NextW(snap, ctypes.byref(e)):
                break
    k32.CloseHandle(snap)
    return pids


hits = find_pid("fxgame.exe")
if not hits:
    print("fxgame.exe not running")
    sys.exit(1)
pid = hits[0][0]
print("== target fxgame.exe pid=%d (ppid=%d) ==" % (pid, hits[0][1]))

# --- 1. escalating OpenProcess ----------------------------------------
tests = [
    ("QUERY_LIMITED", PROCESS_QUERY_LIMITED_INFORMATION),
    ("QUERY_INFO", PROCESS_QUERY_INFORMATION),
    ("VM_READ", PROCESS_VM_READ),
    ("VM_READ|VM_WRITE|VM_OPERATION", PROCESS_VM_READ | PROCESS_VM_WRITE | PROCESS_VM_OPERATION),
    ("PROCESS_ALL_ACCESS", PROCESS_ALL_ACCESS),
]
h = None
for label, acc in tests:
    hh = k32.OpenProcess(acc, False, pid)
    err = ctypes.get_last_error()
    ok = bool(hh)
    print("OpenProcess %-32s -> %s (gle=%d)" % (label, "OK" if ok else "DENIED", 0 if ok else err))
    if ok:
        if h is None:
            h = hh  # keep the strongest-so-far simple handle (VM_READ is enough below)
        elif acc == PROCESS_ALL_ACCESS:
            k32.CloseHandle(hh)
if not h:
    print("cannot open at all; abort")
    sys.exit(2)

# --- 2. modules --------------------------------------------------------
LIST_MODULES_ALL = 0x03
arr = (wt.HMODULE * 2048)()
need = wt.DWORD()


class MODULEINFO(ctypes.Structure):
    _fields_ = [("lpBaseOfDll", ctypes.c_void_p), ("SizeOfImage", wt.DWORD), ("EntryPoint", ctypes.c_void_p)]


if not psapi.EnumProcessModulesEx(h, arr, ctypes.sizeof(arr), ctypes.byref(need), LIST_MODULES_ALL):
    print("EnumProcessModulesEx failed gle=%d" % ctypes.get_last_error())
    sys.exit(3)
count = need.value // ctypes.sizeof(wt.HMODULE)
mods = []
for i in range(count):
    name = ctypes.create_unicode_buffer(260)
    psapi.GetModuleBaseNameW(h, arr[i], name, 260)
    fn = ctypes.create_unicode_buffer(1024)
    psapi.GetModuleFileNameExW(h, arr[i], fn, 1024)
    mi = MODULEINFO()
    psapi.GetModuleInformation(h, arr[i], ctypes.byref(mi), ctypes.sizeof(mi))
    mods.append((name.value, fn.value, mi.lpBaseOfDll or 0, mi.SizeOfImage, mi.EntryPoint or 0))

print("\n== modules: %d loaded ==" % len(mods))
for name, fn, base, size, ep in mods:
    tag = ""
    low = name.lower()
    if any(k in low for k in (".sn", "spdc", "protect", "sprotect", "vmp", "sina", "tenpay", "tp3", "guard")):
        tag = "   <== protection-related"
    print("  %-24s base=0x%012X size=0x%-8X %s%s" % (name, base, size, fn, tag))

tgt = next((m for m in mods if m[0].lower() == "fxgame.exe"), None)
if not tgt:
    print("fxgame module not enumerated")
    sys.exit(4)
_, fpath, base, imgsize, ep = tgt
print("\nfxgame.exe image: %s base=0x%X size=0x%X entrypoint RVA=0x%X" % (fpath, base, imgsize, ep - base))


# --- helpers -----------------------------------------------------------
def rpm(addr, n):
    buf = ctypes.create_string_buffer(n)
    got = ctypes.c_size_t()
    if k32.ReadProcessMemory(h, ctypes.c_void_p(addr), buf, n, ctypes.byref(got)):
        return buf.raw[:got.value]
    got2 = ctypes.c_size_t()
    # retry afterVirtualQuery for guard pages: just report failure
    return None


class MBI(ctypes.Structure):
    _fields_ = [("BaseAddress", ctypes.c_void_p), ("AllocationBase", ctypes.c_void_p),
                ("AllocationProtect", wt.DWORD), ("PartitionId", wt.WORD),
                ("RegionSize", ctypes.c_size_t), ("State", wt.DWORD),
                ("Protect", wt.DWORD), ("Type", wt.DWORD)]


def vq(addr):
    m = MBI()
    if k32.VirtualQueryEx(h, ctypes.c_void_p(addr), ctypes.byref(m), ctypes.sizeof(m)):
        return m
    return None


# --- 3. in-memory PE section table ------------------------------------
hdr = rpm(base, 0x1000)
if not hdr:
    print("cannot read PE headers -> anti-dump (header wiped/unmapped)")
    sys.exit(5)
e_lfanew = struct.unpack_from("<I", hdr, 0x3C)[0]
if hdr[e_lfanew:e_lfanew + 4] != b"PE\0\0":
    print("PE signature invalid in memory (anti-dump?)")
    sys.exit(6)
nsec, optsize = struct.unpack_from("<H", hdr, e_lfanew + 6)[0], struct.unpack_from("<H", hdr, e_lfanew + 20)[0]
magic = struct.unpack_from("<H", hdr, e_lfanew + 24)[0]
sect_off = e_lfanew + 24 + optsize
print("\n== in-memory section table (%d sections, optmagic=0x%X) ==" % (nsec, magic))
secs = []
for i in range(nsec):
    s = hdr[sect_off + 40 * i: sect_off + 40 * (i + 1)]
    nm = s[:8].rstrip(b"\0").decode("latin1", "replace")
    vsize, va, rsize = struct.unpack_from("<III", s, 8)
    chars = struct.unpack_from("<I", s, 36)[0]
    m = vq(base + va)
    prot = page_str(m.Protect) if m else "?"
    state = {0x1000: "COMMIT", 0x2000: "RESERVED", 0x10000: "FREE"}.get(m.State if m else 0, hex(m.State if m else 0))
    # readability + first bytes
    first = rpm(base + va, 16)
    prev = " ".join("%02X" % b for b in (first or b"")[:8]) if first else "(unreadable)"
    ep_mark = " <== ENTRYPOINT" if (base + va) <= ep < (base + va + max(vsize, 1)) else ""
    print("  %-8s VA=0x%06X VSZ=0x%-6X %s state=%s first=%s%s" % (nm, va, vsize, prot, state, prev, ep_mark))
    secs.append((nm, va, vsize))

# --- 5. .text readability detail --------------------------------------
txt = next((s for s in secs if s[0].startswith(".text")), None)
if txt:
    nm, va, vsz = txt
    chunk = rpm(base + va + 0x1000, 4096)
    if chunk:
        nz = sum(1 for b in chunk if b)
        print("\n.text+0x1000 read OK: 4096 bytes, nonzero=%d (decrypted code in place -> runtime-readable)" % nz)
    else:
        print("\n.text unreadable at runtime")

# --- anti-debug quick check -------------------------------------------
dp = ctypes.c_void_p()
status = ntdll.NtQueryInformationProcess(h, 7, ctypes.byref(dp), ctypes.sizeof(dp), None)  # ProcessDebugPort
print("\nNtQueryInformationProcess(ProcessDebugPort) status=0x%08X value=0x%X" % (status & 0xFFFFFFFF, dp.value or 0))
