def island_matrix_counter(matrix: list[list[str]]) -> int:
    seen = set()
    count = 0
    for r, row in enumerate(matrix):
        for c, cell in enumerate(row):
            if cell != "1" or (r, c) in seen:
                continue
            count += 1
            seen.add((r, c))
            stack = [(r, c)]
            while stack:
                y, x = stack.pop()
                for ny, nx in ((y + 1, x), (y - 1, x), (y, x + 1), (y, x - 1)):
                    if (0 <= ny < len(matrix) and 0 <= nx < len(matrix[ny])
                            and matrix[ny][nx] == "1" and (ny, nx) not in seen):
                        seen.add((ny, nx))
                        stack.append((ny, nx))
    return count
