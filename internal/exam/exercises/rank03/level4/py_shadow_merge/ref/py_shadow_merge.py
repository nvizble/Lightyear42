def shadow_merge(list1: list[int], list2: list[int]) -> list[int]:
    merged = []
    i = j = 0
    while i < len(list1) and j < len(list2):
        if list1[i] <= list2[j]:
            merged.append(list1[i])
            i += 1
        else:
            merged.append(list2[j])
            j += 1
    return merged + list1[i:] + list2[j:]
