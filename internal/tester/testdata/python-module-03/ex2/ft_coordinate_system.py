import math


def get_player_pos() -> tuple[float, float, float]:
    while True:
        line = input("Enter new coordinates as floats in format 'x,y,z': ")
        parts = line.split(",")
        try:
            x, y, z = parts
        except ValueError:
            print("Invalid syntax")
            continue
        values: list[float] = []
        for p in parts:
            try:
                values.append(float(p))
            except ValueError as e:
                print(f"Error on parameter '{p.strip()}': {e}")
                break
        else:
            return (values[0], values[1], values[2])


def distance(a: tuple[float, float, float],
             b: tuple[float, float, float]) -> float:
    return math.sqrt((b[0] - a[0]) ** 2 + (b[1] - a[1]) ** 2
                     + (b[2] - a[2]) ** 2)


def main() -> None:
    print("=== Game Coordinate System ===")
    print("Get a first set of coordinates")
    first = get_player_pos()
    print(f"Got a first tuple: {first}")
    x, y, z = first
    print(f"It includes: X={x}, Y={y}, Z={z}")
    print(f"Distance to center: {round(distance((0, 0, 0), first), 4)}")
    print("Get a second set of coordinates")
    second = get_player_pos()
    print("Distance between the 2 sets of coordinates: "
          f"{round(distance(first, second), 4)}")


if __name__ == "__main__":
    main()
