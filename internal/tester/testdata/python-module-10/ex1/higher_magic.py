from collections.abc import Callable


def spell_combiner(spell1: Callable, spell2: Callable) -> Callable:
    def combined(target: str, power: int) -> tuple:
        return spell1(target, power), spell2(target, power)
    return combined


def power_amplifier(base_spell: Callable, multiplier: int) -> Callable:
    def amplified(target: str, power: int) -> str:
        return base_spell(target, power * multiplier)
    return amplified


def conditional_caster(condition: Callable, spell: Callable) -> Callable:
    def caster(target: str, power: int) -> str:
        if condition(target, power):
            return spell(target, power)
        return "Spell fizzled"
    return caster


def spell_sequence(spells: list[Callable]) -> Callable:
    def sequence(target: str, power: int) -> list:
        return [spell(target, power) for spell in spells]
    return sequence


def fireball(target: str, power: int) -> str:
    return f"Fireball hits {target} for {power} damage"


def heal(target: str, power: int) -> str:
    return f"Heal restores {target} for {power} HP"


if __name__ == "__main__":
    print("Testing spell combiner...")
    a, b = spell_combiner(fireball, heal)("Dragon", 10)
    print(f"Combined spell result: {a}, {b}")
    print("Testing power amplifier...")
    print(power_amplifier(fireball, 3)("Dragon", 10))
    print("Testing conditional caster...")
    strong = conditional_caster(lambda t, p: p >= 20, fireball)
    print(strong("Goblin", 30), "/", strong("Goblin", 5))
    print("Testing spell sequence...")
    print(spell_sequence([fireball, heal])("Knight", 7))
