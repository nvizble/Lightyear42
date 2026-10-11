from alchemy.elements import create_air
from elements import create_fire
from ..potions import strength_potion


def lead_to_gold() -> str:
    return ("Recipe transmuting Lead to Gold: brew "
            f"'{create_air()}' and '{strength_potion()}' "
            f"mixed with '{create_fire()}'")
