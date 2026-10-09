import sys


def main() -> None:
    if len(sys.argv) != 2:
        print("Usage: ft_ancient_text.py <file>")
        return
    name = sys.argv[1]
    print("=== Cyber Archives Recovery ===")
    print(f"Accessing file '{name}'")
    try:
        f = open(name)
        content = f.read()
        f.close()
    except OSError as e:
        print(f"Error opening file '{name}': {e}")
        return
    print("---")
    print(content, end="")
    print("---")
    print(f"File '{name}' closed.")


if __name__ == "__main__":
    main()
