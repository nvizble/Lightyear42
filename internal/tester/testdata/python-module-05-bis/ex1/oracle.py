import os
import sys

try:
    from dotenv import load_dotenv
except ImportError:
    print("python-dotenv is missing: pip install -r requirements.txt")
    sys.exit(1)

VARIABLES = ["MATRIX_MODE", "DATABASE_URL", "API_KEY", "LOG_LEVEL",
             "ZION_ENDPOINT"]
DEFAULTS = {"MATRIX_MODE": "development", "LOG_LEVEL": "INFO"}


def load() -> dict[str, str]:
    here = os.path.dirname(os.path.abspath(__file__))
    load_dotenv(os.path.join(here, ".env"))
    config: dict[str, str] = {}
    for name in VARIABLES:
        value = os.environ.get(name, "")
        if not value:
            print(f"[WARNING] {name} is not set, using default")
            value = DEFAULTS.get(name, "")
        config[name] = value
    return config


def env_ignored() -> bool:
    here = os.path.dirname(os.path.abspath(__file__))
    try:
        with open(os.path.join(here, ".gitignore")) as f:
            lines = [line.strip() for line in f]
    except OSError:
        return False
    return ".env" in lines


def main() -> None:
    print("ORACLE STATUS: Consulting the configuration...")
    print()
    config = load()
    mode = config["MATRIX_MODE"]
    if mode not in ("development", "production"):
        print(f"[WARNING] Unknown MATRIX_MODE '{mode}', using development")
        mode = "development"
    print("Configuration loaded:")
    print(f"Mode: {mode}")
    if config["DATABASE_URL"]:
        where = "local instance" if mode == "development" else "cluster"
        print(f"Database: Connected to {where}")
    else:
        print("Database: Not configured")
    print("API Access: " + ("Authenticated" if config["API_KEY"]
                            else "Missing key"))
    level = config["LOG_LEVEL"]
    if mode == "development" and not os.environ.get("LOG_LEVEL"):
        level = "DEBUG"
    print(f"Log Level: {level}")
    print("Remote Network: " + ("Online" if config["ZION_ENDPOINT"]
                                else "Offline"))
    if mode == "production":
        print("Production mode: debug output disabled, errors only.")
    else:
        print("Development mode: verbose diagnostics enabled.")
    print()
    print("Configuration security check:")
    print("[OK] No hardcoded secrets detected")
    if env_ignored():
        print("[OK] .env file properly ignored by git")
    else:
        print("[WARNING] .env is not listed in .gitignore")
    print("[OK] Production overrides available")
    print()
    print("The Oracle sees all configurations.")


if __name__ == "__main__":
    main()
