class Plant:
    def __init__(self, name: str, height: float, age: int,
                 growth_rate: float = 0.8) -> None:
        self.name = name
        self.height = height
        self.p_age = age
        self.growth_rate = growth_rate

    def show(self) -> None:
        print(f"{self.name}: {round(self.height, 1)}cm, {self.p_age} days old")

    def grow(self) -> None:
        self.height += self.growth_rate

    def age(self) -> None:
        self.p_age += 1


def main() -> None:
    print("=== Garden Plant Growth ===")
    rose = Plant("Rose", 25.0, 30)
    start = rose.height
    rose.show()
    for day in range(1, 8):
        print(f"=== Day {day} ===")
        rose.grow()
        rose.age()
        rose.show()
    print(f"Growth this week: {round(rose.height - start, 1)}cm")


if __name__ == "__main__":
    main()
