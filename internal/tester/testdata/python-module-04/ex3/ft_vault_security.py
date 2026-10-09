def secure_archive(name: str, action: str = "read",
                   content: str = "") -> tuple[bool, str]:
    try:
        if action == "write":
            with open(name, "w") as f:
                f.write(content)
            return (True, "Content successfully written to file")
        with open(name) as f:
            return (True, f.read())
    except OSError as e:
        return (False, str(e))


def main() -> None:
    print("=== Cyber Archives Security ===")
    print("Using 'secure_archive' to read from a nonexistent file:")
    print(secure_archive("/not/existing/file"))
    print("Using 'secure_archive' to read from an inaccessible file:")
    print(secure_archive("/etc/master.passwd"))
    print("Using 'secure_archive' to read from a regular file:")
    ok, data = secure_archive("ancient_fragment.txt")
    print((ok, data))
    print("Using 'secure_archive' to write previous content to a new file:")
    print(secure_archive("new_archive.txt", "write", data))


if __name__ == "__main__":
    main()
