# -*- coding: utf-8 -*-
"""Round 2: pin down fxgame VM_READ denial - elevated target vs DACL vs stripped handle."""
import ctypes
import ctypes.wintypes as wt
import subprocess

k32 = ctypes.WinDLL("kernel32", use_last_error=True)
advapi32 = ctypes.WinDLL("advapi32", use_last_error=True)

PID = 58840
READ_CONTROL = 0x00020000
MAXIMUM_ALLOWED = 0x02000000
PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
PROCESS_DUP_HANDLE = 0x0040

# --- our own context ----------------------------------------------------
print("== our shell context ==")
out = subprocess.check_output(["whoami"]).decode().strip()
print("user: %s" % out)
out = subprocess.check_output(["whoami", "/groups"], stderr=subprocess.STDOUT).decode("gbk", "replace")
for line in out.splitlines():
    if " Mandatory Label" in line or "强制标签" in line:
        print("integrity: %s" % line.split()[-1].strip() if line.split() else line)

# --- fxgame cmdline + parent via WMI (no process open needed) -----------
q = "SELECT ProcessId,ParentProcessId,CommandLine FROM Win32_Process WHERE ProcessId=%d OR ProcessId=63328" % PID
out = subprocess.check_output(["powershell", "-NoProfile", "-Command",
                               "Get-CimInstance Win32_Process | Where-Object {$_.ProcessId -in %d,63328} | "
                               "Select-Object ProcessId,Name,CommandLine | Format-List" % PID], stderr=subprocess.STDOUT).decode("gbk", "replace")
print("\n== process info via WMI ==")
print(out.strip())

# --- handle-right probes -------------------------------------------------
print("== handle probes on fxgame ==")
for label, acc in [("READ_CONTROL", READ_CONTROL),
                   ("MAXIMUM_ALLOWED", MAXIMUM_ALLOWED),
                   ("PROCESS_DUP_HANDLE", PROCESS_DUP_HANDLE)]:
    hh = k32.OpenProcess(acc, False, PID)
    err = ctypes.get_last_error()
    if hh:
        print("OpenProcess(%s) -> OK handle=0x%X" % (label, hh))
        if acc == MAXIMUM_ALLOWED:
            # test what we actually got: try GetSecurityInfo (READ_CONTROL bit)
            psid = ctypes.c_void_p()
            ns = wt.DWORD()
            st = advapi32.GetSecurityInfo(hh, 6, 4, ctypes.byref(psid), None, None, None, ctypes.byref(ns))  # OWNER_DACL_INFORMATION
            print("  GetSecurityInfo(owner) -> %s gle=%d" % ("OK" if st == 0 else "fail", ctypes.get_last_error()))
            if st == 0:
                slen = ctypes.wintypes if False else None
                buf = ctypes.create_unicode_buffer(256)
                dom = ctypes.create_unicode_buffer(256)
                use = wt.DWORD()
                advapi32.LookupAccountSidW(None, psid, buf, ctypes.byref(wt.DWORD(256)), dom, ctypes.byref(wt.DWORD(256)), ctypes.byref(use))
                print("  owner: %s\\%s" % (dom.value, buf.value))
        k32.CloseHandle(hh)
    else:
        print("OpenProcess(%s) -> DENIED gle=%d" % (label, err))
