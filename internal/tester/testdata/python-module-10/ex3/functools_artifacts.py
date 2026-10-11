import functools
import operator
from collections.abc import Callable
from typing import Any


def spell_reducer(spells: list[int], operation: str) -> int:
    operations: dict[str, Callable[[int, int], int]] = {
        "add": operator.add,
        "multiply": operator.mul,
        "max": max,
        "min": min,
    }
    if operation not in operations:
        raise ValueError(f"Unknown operation: {operation}")
    if not spells:
        return 0
    return functools.reduce(operations[operation], spells)


def partial_enchanter(base_enchantment: Callable) -> dict[str, Callable]:
    return {
        element: functools.partial(base_enchantment, 50, element)
        for element in ("fire", "ice", "lightning")
    }


@functools.lru_cache(maxsize=None)
def memoized_fibonacci(n: int) -> int:
    if n < 2:
        return n
    return memoized_fibonacci(n - 1) + memoized_fibonacci(n - 2)


def spell_dispatcher() -> Callable[[Any], str]:
    @functools.singledispatch
    def cast(spell: Any) -> str:
        return "Unknown spell type"

    @cast.register
    def _(spell: int) -> str:
        return f"Damage spell: {spell} damage"

    @cast.register
    def _(spell: str) -> str:
        return f"Enchantment: {spell}"

    @cast.register
    def _(spell: list) -> str:
        return f"Multi-cast: {len(spell)} spells"
    return cast


def enchant(power: int, element: str, target: str) -> str:
    return f"{element} enchantment of {power} power on {target}"


if __name__ == "__main__":
    print("Testing spell reducer...")
    powers = [10, 20, 30, 40]
    print(f"Sum: {spell_reducer(powers, 'add')}")
    print(f"Product: {spell_reducer(powers, 'multiply')}")
    print(f"Max: {spell_reducer(powers, 'max')}")
    try:
        spell_reducer(powers, "divide")
    except ValueError as e:
        print(f"Error: {e}")
    print("Testing partial enchanter...")
    for version in partial_enchanter(enchant).values():
        print(version(target="Sword"))
    print("Testing memoized fibonacci...")
    for n in (0, 1, 10, 15):
        print(f"Fib({n}): {memoized_fibonacci(n)}")
    print(memoized_fibonacci.cache_info())
    print("Testing spell dispatcher...")
    dispatch = spell_dispatcher()
    for spell in (42, "fireball", ["a", "b", "c"], 3.5):
        print(dispatch(spell))
