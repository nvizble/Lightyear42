def generate_spiral(n: int) -> list[list[int]]:
    m = [[0] * n for _ in range(n)]
    r, c, dr, dc = 0, 0, 0, 1
    for k in range(1, n * n + 1):
        m[r][c] = k
        nr, nc = r + dr, c + dc
        if not (0 <= nr < n and 0 <= nc < n) or m[nr][nc]:
            dr, dc = dc, -dr
            nr, nc = r + dr, c + dc
        r, c = nr, nc
    return m
