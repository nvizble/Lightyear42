from ex0 import AquaFactory, CreatureFactory, FlameFactory


def test_factory(factory: CreatureFactory) -> None:
    print("Testing factory")
    for creature in (factory.create_base(), factory.create_evolved()):
        print(creature.describe())
        print(creature.attack())


def battle(first: CreatureFactory, second: CreatureFactory) -> None:
    print("Testing battle")
    a = first.create_base()
    b = second.create_base()
    print(a.describe())
    print("vs.")
    print(b.describe())
    print("fight!")
    print(a.attack())
    print(b.attack())


if __name__ == "__main__":
    flame = FlameFactory()
    aqua = AquaFactory()
    test_factory(flame)
    test_factory(aqua)
    battle(flame, aqua)
