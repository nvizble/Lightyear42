from .dark_spellbook import dark_spell_allowed_ingredients


def dark_validate_ingredients(ingredients: str) -> str:
    lowered = ingredients.lower()
    ok = any(item in lowered for item in dark_spell_allowed_ingredients())
    return f"{ingredients} - {'VALID' if ok else 'INVALID'}"
