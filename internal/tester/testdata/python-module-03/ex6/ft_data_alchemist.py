import random


def main() -> None:
    print("=== Game Data Alchemist ===")
    players = ["Alice", "bob", "Charlie", "dylan", "Emma", "Gregory", "john",
               "kevin", "Liam"]
    print(f"Initial list of players: {players}")
    capitalized = [name.capitalize() for name in players]
    print(f"New list with all names capitalized: {capitalized}")
    only = [name for name in players if name == name.capitalize()]
    print(f"New list of capitalized names only: {only}")
    scores = {name: random.randint(0, 1000) for name in capitalized}
    print(f"Score dict: {scores}")
    average = sum(scores.values()) / len(scores)
    print(f"Score average is {round(average, 2)}")
    high = {name: s for name, s in scores.items() if s > average}
    print(f"High scores: {high}")


if __name__ == "__main__":
    main()
