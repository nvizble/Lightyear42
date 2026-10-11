from elements import create_fire, create_water
from .elements import create_air, create_earth


def healing_potion() -> str:
    return (f"Healing potion brewed with '{create_earth()}'"
            f" and '{create_air()}'")


def strength_potion() -> str:
    return (f"Strength potion brewed with '{create_fire()}'"
            f" and '{create_water()}'")
