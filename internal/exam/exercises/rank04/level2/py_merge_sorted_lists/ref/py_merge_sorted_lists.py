def _merge(a: list[int], b: list[int]) -> list[int]:
    out = []
    i = j = 0
    while i < len(a) and j < len(b):
        if a[i] <= b[j]:
            out.append(a[i])
            i += 1
        else:
            out.append(b[j])
            j += 1
    out.extend(a[i:])
    out.extend(b[j:])
    return out


def merge_sorted_lists(lists: list[list[int]]) -> list[int]:
    lists = [lst for lst in lists if lst]
    if not lists:
        return []
    while len(lists) > 1:
        pairs = []
        for i in range(0, len(lists) - 1, 2):
            pairs.append(_merge(lists[i], lists[i + 1]))
        if len(lists) % 2:
            pairs.append(lists[-1])
        lists = pairs
    return list(lists[0])
