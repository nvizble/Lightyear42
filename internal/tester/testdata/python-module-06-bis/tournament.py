from ex3 import CreatureFactory, FlameFactory, TransformCreatureFactory
from ex4 import (AggressiveStrategy, BattleStrategy, NormalStrategy,
                 StrategyError)

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
    transform = TransformCreatureFactory()
    normal = NormalStrategy()
    aggressive = AggressiveStrategy()
    print("Tournament 0 (basic)")
    print("[ (Flameling+Normal), (Shiftling+Aggressive) ]")
    battle([(flame, normal), (transform, aggressive)])
    print()
    print("Tournament 1 (error)")
    print("[ (Shiftling+Aggressive), (Flameling+Aggressive) ]")
    battle([(transform, aggressive), (flame, aggressive)])
