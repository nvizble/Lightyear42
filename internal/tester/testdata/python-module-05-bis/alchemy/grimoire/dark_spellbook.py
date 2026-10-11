from .dark_validator import dark_validate_ingredients


def dark_spell_allowed_ingredients() -> list[str]:
    return ["bats", "frogs", "arsenic", "eyeball"]


def dark_spell_record(spell_name: str, ingredients: str) -> str:
    result = dark_validate_ingredients(ingredients)
    if result.endswith("INVALID"):
        return f"Dark spell rejected: {spell_name} ({result})"
    return f"Dark spell recorded: {spell_name} ({result})"
