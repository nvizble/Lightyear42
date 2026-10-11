from .light_spellbook import light_spell_allowed_ingredients


def validate_ingredients(ingredients: str) -> str:
    low = ingredients.lower()
    if any(i in low for i in light_spell_allowed_ingredients()):
        return f"{ingredients} - VALID"
    return f"{ingredients} - INVALID"
