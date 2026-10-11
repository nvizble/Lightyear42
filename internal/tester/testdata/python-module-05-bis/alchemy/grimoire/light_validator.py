def validate_ingredients(ingredients: str) -> str:
    from .light_spellbook import light_spell_allowed_ingredients
    allowed = light_spell_allowed_ingredients()
    lowered = ingredients.lower()
    ok = any(item in lowered for item in allowed)
    return f"{ingredients} - {'VALID' if ok else 'INVALID'}"
