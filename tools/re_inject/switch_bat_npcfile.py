# 切换 start-9yin.bat 的 NINEYIN_REPLAY_NPC_FILE → 类型扫描包（字节级，保 GBK+CRLF）
import sys

p = r'F:/Game/Online/9yin/9yin-server-go/start-9yin.bat'
raw = open(p, 'rb').read()

needle = b'NINEYIN_REPLAY_NPC_FILE='
lines = raw.split(b'\r\n')
hit = 0
for i, l in enumerate(lines):
    if needle in l and not l.lstrip().startswith(b'rem'):
        new = b'set "NINEYIN_REPLAY_NPC_FILE=G:\\pojie\\ai\\9yin-server-go\\tmp\\op28_analysis\\npc_typesweep46.bin"'
        lines[i] = new
        hit += 1
if hit != 1:
    print(f'FATAL: {hit} active lines')
    sys.exit(1)
raw2 = b'\r\n'.join(lines)
open(p, 'wb').write(raw2)
raw3 = open(p, 'rb').read()
assert b'\r\n' in raw3
assert b'npc_typesweep46.bin' in raw3
print('bat switched to typesweep pack')
