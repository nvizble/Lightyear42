def inter(s1: str, s2: str) -> str:
    result = ""
    for c in s1:
        if c in s2 and c not in result:
            result += c
    return result
