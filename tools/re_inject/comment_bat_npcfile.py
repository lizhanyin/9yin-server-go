# 注释掉 start-9yin.bat 的 NINEYIN_REPLAY_NPC_FILE 行（判别实验收官）
p = r'F:/Game/Online/9yin/9yin-server-go/start-9yin.bat'
raw = open(p, 'rb').read()
lines = raw.split(b'\r\n')
note = 'rem 判别实验已收官（原生帧/类型/头标四轮全过），NPC 注入通道停用：'.encode('gbk')
hit = 0
for i, l in enumerate(lines):
    if b'NINEYIN_REPLAY_NPC_FILE=' in l and not l.lstrip().startswith(b'rem'):
        lines[i] = note + l
        hit += 1
assert hit == 1, hit
out = b'\r\n'.join(lines)
out.decode('gbk')  # 编码完整性
open(p, 'wb').write(out)
print('已注释，行数不变')
