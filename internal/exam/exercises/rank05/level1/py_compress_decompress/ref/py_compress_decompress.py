def compress(s: str) -> str:
    out = []
    i = 0
    while i < len(s):
        j = i
        while j < len(s) and s[j] == s[i]:
            j += 1
        out.append(s[i] + (str(j - i) if j - i > 1 else ""))
        i = j
    return "".join(out)


def decompress(s: str) -> str:
    out = []
    i = 0
    while i < len(s):
        c = s[i]
        j = i + 1
        while j < len(s) and s[j].isdigit():
            j += 1
        out.append(c * (int(s[i + 1:j]) if j > i + 1 else 1))
        i = j
    return "".join(out)
