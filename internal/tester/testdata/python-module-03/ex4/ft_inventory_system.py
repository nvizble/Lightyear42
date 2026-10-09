import sys


def parse(args: list[str]) -> dict[str, int]:
    inventory: dict[str, int] = {}
    for arg in args:
        parts = arg.split(":")
        if len(parts) != 2 or parts[0] == "":
            print(f"Error - invalid parameter '{arg}'")
            continue
        name, raw = parts
        if name in inventory.keys():
            print(f"Redundant item '{name}' - discarding")
            continue
        try:
            inventory.update({name: int(raw)})
        except ValueError as e:
            print(f"Quantity error for '{name}': {e}")
    return inventory


def main() -> None:
    print("=== Inventory System Analysis ===")
    inventory = parse(sys.argv[1:])
    print(f"Got inventory: {inventory}")
    items = list(inventory.keys())
    print(f"Item list: {items}")
    total = sum(inventory.values())
    print(f"Total quantity of the {len(items)} items: {total}")
    for name in items:
        share = inventory[name] / total * 100 if total else 0.0
        print(f"Item {name} represents {round(share, 1)}%")
    if len(items) > 0:
        most = items[0]
        least = items[0]
        for name in items:
            if inventory[name] > inventory[most]:
                most = name
            if inventory[name] < inventory[least]:
                least = name
        print(f"Item most abundant: {most} with quantity {inventory[most]}")
        print(f"Item least abundant: {least} with quantity "
              f"{inventory[least]}")
    inventory.update({"magic_item": 1})
    print(f"Updated inventory: {inventory}")


if __name__ == "__main__":
    main()
