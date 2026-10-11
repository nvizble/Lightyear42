def artifact_sorter(artifacts: list[dict]) -> list[dict]:
    return sorted(artifacts, key=lambda a: a["power"], reverse=True)


def power_filter(mages: list[dict], min_power: int) -> list[dict]:
    return list(filter(lambda m: m["power"] >= min_power, mages))


def spell_transformer(spells: list[str]) -> list[str]:
    return list(map(lambda s: "* " + s + " *", spells))


def mage_stats(mages: list[dict]) -> dict:
    return {
        "max_power": max(mages, key=lambda m: m["power"])["power"],
        "min_power": min(mages, key=lambda m: m["power"])["power"],
        "avg_power": round(sum(map(lambda m: m["power"], mages))
                           / len(mages), 2),
    }


if __name__ == "__main__":
    artifacts = [
        {"name": "Crystal Orb", "power": 85, "type": "orb"},
        {"name": "Fire Staff", "power": 92, "type": "staff"},
    ]
    print("Testing artifact sorter...")
    first, second = artifact_sorter(artifacts)
    print(f"{first['name']} ({first['power']} power) comes before "
          f"{second['name']} ({second['power']} power)")
    print("Testing spell transformer...")
    print(" ".join(spell_transformer(["fireball", "heal", "shield"])))
    mages = [
        {"name": "Alex", "power": 70, "element": "fire"},
        {"name": "Riley", "power": 45, "element": "ice"},
    ]
    print("Testing power filter...")
    print(power_filter(mages, 50))
    print("Testing mage stats...")
    print(mage_stats(mages))
