def hidenp(small: str, big: str) -> bool:
    i = 0
    for c in big:
        if i < len(small) and c == small[i]:
            i += 1
    return i == len(small)
