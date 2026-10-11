from abc import ABC, abstractmethod

from ex3.capability import TransformCapability
from ex3.creature import Creature


class StrategyError(Exception):
    pass


class BattleStrategy(ABC):
    @abstractmethod
    def is_valid(self, creature: Creature) -> bool:
        ...

    @abstractmethod
    def act(self, creature: Creature) -> None:
        ...


class NormalStrategy(BattleStrategy):
    def is_valid(self, creature: Creature) -> bool:
        return True

    def act(self, creature: Creature) -> None:
        print(creature.attack())


class AggressiveStrategy(BattleStrategy):
    def is_valid(self, creature: Creature) -> bool:
        return isinstance(creature, TransformCapability)

    def act(self, creature: Creature) -> None:
        if not isinstance(creature, TransformCapability):
            raise StrategyError(f"Invalid Creature '{creature.name}' "
                                "for this aggressive strategy")
        print(creature.transform())
        print(creature.attack())
        print(creature.revert())
