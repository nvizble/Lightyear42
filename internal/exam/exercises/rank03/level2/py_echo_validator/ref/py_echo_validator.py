def echo_validator(text: str) -> bool:
    letters = [c.lower() for c in text if c.isalpha()]
    return len(letters) > 0 and letters == letters[::-1]
