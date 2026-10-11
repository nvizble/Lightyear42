import os
import site
import sys


def in_venv() -> bool:
    return sys.prefix != sys.base_prefix


def site_packages() -> str:
    paths = site.getsitepackages()
    return paths[0] if paths else "unknown"


def show_open() -> None:
    print("LABORATORY STATUS: You are working in the open")
    print(f"Current Python: {sys.executable}")
    print("Virtual Environment: None detected")
    print()
    print("WARNING: You are in the global environment!")
    print("Every reagent you install here leaks into the whole system.")
    print(f"Global reagent path: {site_packages()}")
    print()
    print("To seal the laboratory, run:")
    print("python3 -m venv lab_env")
    print("source lab_env/bin/activate  # On Unix")
    print("lab_env\\Scripts\\activate  # On Windows")
    print()
    print("Then run this program again.")


def show_sealed() -> None:
    print("LABORATORY STATUS: The laboratory is sealed")
    print(f"Current Python: {sys.executable}")
    print(f"Virtual Environment: {os.path.basename(sys.prefix)}")
    print(f"Environment Path: {sys.prefix}")
    print()
    print("SUCCESS: You are working in an isolated environment!")
    print("Safe to install reagents without affecting")
    print("the global system.")
    print()
    print("Reagent installation path:")
    print(site_packages())
    print(f"Global installation base: {sys.base_prefix}")


def main() -> None:
    if in_venv():
        show_sealed()
    else:
        show_open()


if __name__ == "__main__":
    main()
