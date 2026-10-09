class Plant:
    name: str = ""
    height: int = 0
    age: int = 0

    def show(self) -> None:
        print(f"{self.name}: {self.height}cm, {self.age} days old")


def main() -> None:
    print("=== Garden Plant Registry ===")
    for name, height, age in (("Rose", 25, 30), ("Sunflower", 80, 45),
                              ("Cactus", 15, 120)):
        plant = Plant()
        plant.name = name
        plant.height = height
        plant.age = age
        plant.show()


if __name__ == "__main__":
    main()
