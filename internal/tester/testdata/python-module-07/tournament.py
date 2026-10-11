from ex0 import AquaFactory, CreatureFactory, FlameFactory
from ex1 import HealingCreatureFactory, TransformCreatureFactory
from ex2 import (AggressiveStrategy, BattleStrategy, DefensiveStrategy,
                 NormalStrategy, StrategyError)

Opponent = tuple[CreatureFactory, BattleStrategy]


def battle(opponents: list[Opponent]) -> None:
    print("*** Tournament ***")
    print(f"{len(opponents)} opponents involved")
    fighters = [(f.create_base(), s) for f, s in opponents]
    try:
        for i, (a, sa) in enumerate(fighters):
            for b, sb in fighters[i + 1:]:
                print("* Battle *")
                print(a.describe())
                print("vs.")
                print(b.describe())
                print("now fight!")
                sa.act(a)
                sb.act(b)
    except StrategyError as e:
        print(f"Battle error, aborting tournament: {e}")


if __name__ == "__main__":
    flame = FlameFactory()
    aqua = AquaFactory()
    healing = HealingCreatureFactory()
    transform = TransformCreatureFactory()
    normal = NormalStrategy()
    aggressive = AggressiveStrategy()
    defensive = DefensiveStrategy()
    print("Tournament 0 (basic)")
    print("[ (Flameling+Normal), (Healing+Defensive) ]")
    battle([(flame, normal), (healing, defensive)])
    print()
    print("Tournament 1 (error)")
    print("[ (Flameling+Aggressive), (Healing+Defensive) ]")
    battle([(flame, aggressive), (healing, defensive)])
    print()
    print("Tournament 2 (multiple)")
    print("[ (Aquabub+Normal), (Healing+Defensive), (Transform+Aggressive) ]")
    battle([(aqua, normal), (healing, defensive), (transform, aggressive)])
