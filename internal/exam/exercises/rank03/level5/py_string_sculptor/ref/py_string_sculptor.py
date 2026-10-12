def string_sculptor(text: str) -> str:
    out = []
    upper = False
    for c in text:
        if c == " ":
            upper = False
        elif c.isalpha():
            c = c.upper() if upper else c.lower()
            upper = not upper
        out.append(c)
    return "".join(out)
