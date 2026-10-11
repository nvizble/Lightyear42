import importlib

REAGENTS = {
    "pandas": "Data manipulation ready",
    "numpy": "Numerical computation ready",
}


def check(name: str, role: str) -> bool:
    try:
        module = importlib.import_module(name)
    except ImportError:
        print(f"[MISSING] {name} - not installed")
        return False
    version = getattr(module, "__version__", "unknown")
    print(f"[OK] {name} ({version}) - {role}")
    return True


def main() -> None:
    print("REAGENT STATUS: Loading reagents...")
    print()
    print("Checking dependencies:")
    missing = [n for n, role in REAGENTS.items() if not check(n, role)]
    print()
    print("pip vs Poetry:")
    print("requirements.txt declares what to install; "
          "pip resolves it at install time.")
    print("pyproject.toml declares the same constraints, and Poetry pins "
          "the resolved")
    print("versions in poetry.lock so every install is identical.")
    print()
    if missing:
        print("Some reagents are missing. Install them with:")
        print("  pip install -r requirements.txt")
        print(f"  pip install {' '.join(missing)}")
        print("or with Poetry:")
        print("  poetry install")
        print(f"  poetry add {' '.join(missing)}")
    else:
        print("All reagents present and accounted for.")


if __name__ == "__main__":
    main()
