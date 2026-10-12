def anagram(s1: str, s2: str) -> bool:
    def count(s):
        counts = {}
        for c in s.lower():
            if c != " ":
                counts[c] = counts.get(c, 0) + 1
        return counts
    return count(s1) == count(s2)
