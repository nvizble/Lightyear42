def list_intersection_finder(lists: list[list[int]]) -> list[int]:
    if not lists:
        return []
    common = set(lists[0])
    for other in lists[1:]:
        common &= set(other)
    return sorted(common)
