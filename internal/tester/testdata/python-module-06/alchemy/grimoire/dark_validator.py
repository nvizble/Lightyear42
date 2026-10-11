from .dark_spellbook import dark_spell_allowed_ingredients


def validate_ingredients(ingredients: str) -> str:
    low = ingredients.lower()
    if any(i in low for i in dark_spell_allowed_ingredients()):
        return f"{ingredients} - VALID"
    return f"{ingredients} - INVALID"
