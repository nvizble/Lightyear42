def word_ladder(start: str, end: str, sentence: list[str]) -> int:
    words = set(sentence)
    if end not in words:
        return 0
    letters = {c for w in words for c in w}
    frontier, seen, length = [start], {start}, 1
    while frontier:
        length += 1
        nxt = []
        for word in frontier:
            for i in range(len(word)):
                for c in letters:
                    cand = word[:i] + c + word[i + 1:]
                    if cand in words and cand not in seen:
                        if cand == end:
                            return length
                        seen.add(cand)
                        nxt.append(cand)
        frontier = nxt
    return 0
