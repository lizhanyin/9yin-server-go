# 设置 start-9yin.bat 的 NINEYIN_REPLAY_NPC_FILE → 指定包（整行替换，容忍既有损坏内容）
import sys

BAT = r'F:/Game/Online/9yin/9yin-server-go/start-9yin.bat'
NEW = b'set "NINEYIN_REPLAY_NPC_FILE=G:\\pojie\\ai\\9yin-server-go\\tmp\\op28_analysis\\' + sys.argv[1].encode() + b'"'

raw = open(BAT, 'rb').read()
lines = raw.split(b'\r\n')
hit = 0
for i, l in enumerate(lines):
    if b'NINEYIN_REPLAY_NPC_FILE=' in l and not l.lstrip().startswith(b'rem'):
        lines[i] = NEW
        hit += 1
if hit != 1:
    print(f'FATAL: {hit} active lines')
    sys.exit(1)
out = b'\r\n'.join(lines)
out.decode('gbk')  # 编码完整性
assert out.count(b'\r\n') == raw.count(b'\r\n') or True
open(BAT, 'wb').write(out)
print('written:', NEW.decode())
