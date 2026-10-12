DIRECTIONS = [
    (1, 0, "H"), (-1, 0, "H-"), (0, 1, "V"), (0, -1, "V-"),
    (1, 1, "D1"), (-1, -1, "D1-"), (-1, 1, "D2"), (1, -1, "D2-"),
]


def prism_detector(grid: list[str], pattern: str):
    found = []
    if not grid or not pattern:
        return found
    rows, n = len(grid), len(pattern)
    for y in range(rows):
        for x in range(len(grid[y])):
            for dx, dy, code in DIRECTIONS:
                ex, ey = x + dx * (n - 1), y + dy * (n - 1)
                if not (0 <= ey < rows and 0 <= ex < len(grid[y])):
                    continue
                if all(grid[y + dy * i][x + dx * i] == pattern[i]
                       for i in range(n)):
                    found.append((x, y, code))
    return found
