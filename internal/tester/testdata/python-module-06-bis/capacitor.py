from ex3 import TransformCreatureFactory
from ex3.capability import TransformCapability
from ex3.creature import Creature


def show_transform(label: str, creature: Creature) -> None:
    print(f"{label}:")
    print(creature.describe())
    print(creature.attack())
    if isinstance(creature, TransformCapability):
        print(creature.transform())
        print(creature.attack())
        print(creature.revert())


if __name__ == "__main__":
    print("Testing Creature with transform capability")
    transform = TransformCreatureFactory()
    show_transform("base", transform.create_base())
    show_transform("evolved", transform.create_evolved())
