def palindrome_partitioner(s: str) -> int:
    n = len(s)
    if n < 2:
        return 0
    pal = [[False] * n for _ in range(n)]
    cuts = [0] * n
    for i in range(n):
        best = i
        for j in range(i + 1):
            if s[j] == s[i] and (i - j < 2 or pal[j + 1][i - 1]):
                pal[j][i] = True
                best = 0 if j == 0 else min(best, cuts[j - 1] + 1)
        cuts[i] = best
    return cuts[-1]
