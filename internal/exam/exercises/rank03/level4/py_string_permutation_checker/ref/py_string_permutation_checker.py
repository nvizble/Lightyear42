def string_permutation_checker(s1: str, s2: str) -> bool:
    counts = {}
    for c in s1:
        counts[c] = counts.get(c, 0) + 1
    for c in s2:
        counts[c] = counts.get(c, 0) - 1
    return all(n == 0 for n in counts.values())
