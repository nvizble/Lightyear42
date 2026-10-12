def whisper_cipher(text: str, shift: int) -> str:
    out = []
    for c in text:
        if "a" <= c <= "z":
            c = chr((ord(c) - ord("a") + shift) % 26 + ord("a"))
        elif "A" <= c <= "Z":
            c = chr((ord(c) - ord("A") + shift) % 26 + ord("A"))
        out.append(c)
    return "".join(out)
