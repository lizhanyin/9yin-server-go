# 修 F: 部署 start-9yin.bat 的 NINEYIN_REPLAY_NPC_FILE：'1'（错的开关用法）→ 预验证包路径
# 字节级替换保 GBK+CRLF（bat 铁律：Edit 工具会写 UTF-8/LF，禁用）
import sys

p = r'F:/Game/Online/9yin/9yin-server-go/start-9yin.bat'
raw = open(p, 'rb').read()

old = b'set "NINEYIN_REPLAY_NPC_FILE=1"'
new = b'set "NINEYIN_REPLAY_NPC_FILE=G:\\pojie\\ai\\9yin-server-go\\tmp\\npc_native46_replay.bin"'

cnt = raw.count(old)
if cnt != 1:
    print(f'FATAL: expected 1 occurrence, found {cnt}')
    sys.exit(1)

raw2 = raw.replace(old, new)
open(p, 'wb').write(raw2)

# 复核
raw3 = open(p, 'rb').read()
assert b'\r\n' in raw3, 'CRLF lost!'
lines = [l for l in raw3.split(b'\r\n') if b'NINEYIN_REPLAY_NPC_FILE' in l and not l.lstrip().startswith(b'rem')]
print('OK, now:')
print(lines[0].decode('gbk'))
