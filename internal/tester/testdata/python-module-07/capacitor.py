from ex0.creature import Creature
from ex1 import HealingCreatureFactory, TransformCreatureFactory
from ex1.capability import HealCapability, TransformCapability


def show_healing(label: str, creature: Creature) -> None:
    print(f"{label}:")
    print(creature.describe())
    print(creature.attack())
    if isinstance(creature, HealCapability):
        print(creature.heal())


def show_transform(label: str, creature: Creature) -> None:
    print(f"{label}:")
    print(creature.describe())
    print(creature.attack())
    if isinstance(creature, TransformCapability):
        print(creature.transform())
        print(creature.attack())
        print(creature.revert())


if __name__ == "__main__":
    print("Testing Creature with healing capability")
    healing = HealingCreatureFactory()
    show_healing("base", healing.create_base())
    show_healing("evolved", healing.create_evolved())
    print()
    print("Testing Creature with transform capability")
    transform = TransformCreatureFactory()
    show_transform("base", transform.create_base())
    show_transform("evolved", transform.create_evolved())
