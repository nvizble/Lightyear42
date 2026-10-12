DIGITS = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"


def number_base_converter(number: str, from_base: int, to_base: int) -> str:
    if not (2 <= from_base <= 36 and 2 <= to_base <= 36) or number == "":
        return "ERROR"
    value = 0
    for c in number:
        d = DIGITS.find(c.upper())
        if d < 0 or d >= from_base:
            return "ERROR"
        value = value * from_base + d
    out = ""
    while True:
        value, d = divmod(value, to_base)
        out = DIGITS[d] + out
        if value == 0:
            return out
