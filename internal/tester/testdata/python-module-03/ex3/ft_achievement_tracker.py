import random

ACHIEVEMENTS = [
    "Crafting Genius", "World Savior", "Master Explorer", "Collector Supreme",
    "Untouchable", "Boss Slayer", "Strategist", "Unstoppable", "Speed Runner",
    "Survivor", "Treasure Hunter", "First Steps", "Sharp Mind",
    "Hidden Path Finder",
]


def gen_player_achievements() -> set[str]:
    count = random.randint(5, 9)
    return set(random.sample(ACHIEVEMENTS, count))


def main() -> None:
    print("=== Achievement Tracker System ===")
    names = ["Alice", "Bob", "Charlie", "Dylan"]
    sets: list[set[str]] = []
    for name in names:
        sets.append(gen_player_achievements())
        print(f"Player {name}: {sets[-1]}")
    every: set[str] = set().union(*sets)
    common = set(ACHIEVEMENTS).intersection(*sets)
    print(f"All distinct achievements: {every}")
    print(f"Common achievements: {common}")
    i = 0
    for name in names:
        others: set[str] = set()
        for other in sets[:i] + sets[i + 1:]:
            others = others.union(other)
        print(f"Only {name} has: {sets[i].difference(others)}")
        i += 1
    i = 0
    for name in names:
        print(f"{name} is missing: {set(ACHIEVEMENTS).difference(sets[i])}")
        i += 1


if __name__ == "__main__":
    main()
