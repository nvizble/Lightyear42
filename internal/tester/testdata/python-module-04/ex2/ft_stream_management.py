import sys


def read_file(name: str) -> str | None:
    print(f"Accessing file '{name}'")
    try:
        f = open(name)
        content = f.read()
        f.close()
    except OSError as e:
        print(f"[STDERR] Error opening file '{name}': {e}", file=sys.stderr)
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
        print(f"[STDERR] Error opening file '{name}': {e}", file=sys.stderr)
        print("Data not saved.")
        return
    print(f"Data saved in file '{name}'.")


def main() -> None:
    if len(sys.argv) != 2:
        sys.stderr.write("Usage: ft_stream_management.py <file>\n")
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
    sys.stdout.write("Enter new file name (or empty): ")
    sys.stdout.flush()
    name = sys.stdin.readline().rstrip("\n")
    if name == "":
        print("Not saving data.")
        return
    save_file(name, new)


if __name__ == "__main__":
    main()
