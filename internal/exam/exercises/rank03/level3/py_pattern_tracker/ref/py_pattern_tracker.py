def pattern_tracker(text: str) -> int:
    count = 0
    for a, b in zip(text, text[1:]):
        if a in "012345678" and b in "0123456789" and int(b) == int(a) + 1:
            count += 1
    return count
