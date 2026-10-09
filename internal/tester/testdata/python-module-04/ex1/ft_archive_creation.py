import sys


def read_file(name: str) -> str | None:
    print(f"Accessing file '{name}'")
    try:
        f = open(name)
        content = f.read()
        f.close()
    except OSError as e:
        print(f"Error opening file '{name}': {e}")
        return None
    print("---")
    print(content, end="")
    print("---")
    print(f"File '{name}' closed.")
    return content


def save_file(name: str, content: str) -> None:
    print(f"Saving data to '{name}'")
    try:
        f = open(name, "w")
        f.write(content)
        f.close()
    except OSError as e:
        print(f"Error opening file '{name}': {e}")
        print("Data not saved.")
        return
    print(f"Data saved in file '{name}'.")


def main() -> None:
    if len(sys.argv) != 2:
        print("Usage: ft_archive_creation.py <file>")
        return
    print("=== Cyber Archives Recovery & Preservation ===")
    content = read_file(sys.argv[1])
    if content is None:
        return
    new = "".join(line + "#\n" for line in content.splitlines())
    print("Transform data:")
    print("---")
    print(new, end="")
    print("---")
    name = input("Enter new file name (or empty): ")
    if name == "":
        print("Not saving data.")
        return
    save_file(name, new)


if __name__ == "__main__":
    main()
