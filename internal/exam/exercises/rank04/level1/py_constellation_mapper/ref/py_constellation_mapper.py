def constellation_mapper(stars: list[tuple[int, int]], dim: int) -> list[str]:
    if dim <= 0:
        return []
    grid = [["."] * dim for _ in range(dim)]
    for row, col in stars:
        if 0 <= row < dim and 0 <= col < dim:
            grid[row][col] = "*"
    return ["".join(line) for line in grid]
