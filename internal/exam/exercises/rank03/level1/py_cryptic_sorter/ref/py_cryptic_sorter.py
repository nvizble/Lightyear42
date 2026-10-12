def cryptic_sorter(strings: list[str]) -> list[str]:
    result = []
    for s in strings:
        key = (len(s), s.lower())
        i = len(result)
        while i > 0 and (len(result[i - 1]), result[i - 1].lower()) > key:
            i -= 1
        result.insert(i, s)
    return result
