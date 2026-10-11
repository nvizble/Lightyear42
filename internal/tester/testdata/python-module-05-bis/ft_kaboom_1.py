print("=== Kaboom 1 ===")
print("Access to alchemy/grimoire/dark_spellbook.py directly")
print("Test import now - THIS WILL RAISE AN UNCAUGHT EXCEPTION", flush=True)

from alchemy.grimoire.dark_spellbook import dark_spell_record  # noqa: E402

print(f"Testing record dark spell: {dark_spell_record('Doom', 'bats')}")
