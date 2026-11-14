# 扫 Go 双引号串里的 \f \v \a \b 单转义（疑似误写反斜杠路径的 bug 形）
import re, glob

STR = re.compile(r'"((?:[^"\\]|\\.)*)"')
BAD = re.compile(r'\\[fvab]')

hits = []
for f in glob.glob('cmd/protocol-probe/*.go') + glob.glob('internal/**/*.go', recursive=True):
    if '_test' in f:
        continue
    for i, line in enumerate(open(f, encoding='utf-8', errors='replace'), 1):
        for m in STR.finditer(line):
            s = m.group(1)
            if BAD.search(s):
                hits.append((f, i, s))

for f, i, s in hits:
    print(f'{f}:{i}: {s!r}')
print(len(hits), 'suspects')
