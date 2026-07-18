#!/usr/bin/env python3
"""Collect Intra subject PDF CDN ids for lightyear.

Default: crawl the paginated projects index
  https://projects.intra.42.fr/projects/list
  https://projects.intra.42.fr/projects/list?page=2
…then open each project page and extract
  https://cdn.intra.42.fr/pdf/pdf/<id>/<lang>.subject.pdf

Optional: --slugs-file to scrape only listed slugs.

Output: catalog.json  {"slug": id, ...}

Local tooling only — /tools/ is gitignored from the lightyear repo.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
import time
from pathlib import Path

from playwright.sync_api import sync_playwright

CDN_RE = re.compile(
    r"https://cdn\.intra\.42\.fr/pdf/pdf/(\d+)/([a-z]+)\.subject\.pdf"
)
PROJECT_HREF_RE = re.compile(
    r"""href=["'](?:https://projects\.intra\.42\.fr)?/projects/"""
    r"""([a-zA-Z0-9][a-zA-Z0-9._-]*)/?["']""",
    re.IGNORECASE,
)
SKIP_SLUGS = frozenset(
    {
        "list",
        "search",
        "graph",
        "new",
        "filter",
        "mine",
        "available",
    }
)

HERE = Path(__file__).resolve().parent
DEFAULT_OUT = HERE / "catalog.json"
DEFAULT_PROFILE = HERE / ".pw-profile"
INTRA_SIGNIN = "https://signin.intra.42.fr/users/sign_in"
INTRA_HOME = "https://profile.intra.42.fr/"
PROJECTS_LIST = "https://projects.intra.42.fr/projects/list"
PROJECT_URL = "https://projects.intra.42.fr/projects/{slug}"


def load_slugs(path: Path) -> list[str]:
    slugs: list[str] = []
    for line in path.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        slugs.append(line)
    return slugs


def extract_slugs(html: str) -> list[str]:
    seen: set[str] = set()
    out: list[str] = []
    for slug in PROJECT_HREF_RE.findall(html):
        slug = slug.strip()
        low = slug.lower()
        if low in SKIP_SLUGS or low in seen:
            continue
        seen.add(low)
        out.append(slug)
    return out


def pick_pdf_id(html: str, prefer_lang: str) -> int | None:
    matches = CDN_RE.findall(html)
    if not matches:
        return None
    for pdf_id, lang in matches:
        if prefer_lang and lang == prefer_lang:
            return int(pdf_id)
    return int(matches[0][0])


def wait_for_login_manual() -> None:
    print(
        "\n=== Login manual ===\n"
        "1) Na janela do Chromium, faça login na Intra até ver o perfil/dashboard.\n"
        "2) Volte a ESTE terminal e pressione Enter para começar o scrape.\n"
        "(O script não mexe no browser enquanto espera.)\n",
        flush=True,
    )
    try:
        input("Pressione Enter quando estiver autenticado… ")
    except EOFError as exc:
        raise SystemExit("Login cancelado.") from exc
    print("A continuar…\n", flush=True)


def already_logged_in(page) -> bool:
    try:
        page.goto(INTRA_HOME, wait_until="domcontentloaded", timeout=30_000)
        url = (page.url or "").lower()
        if "signin" in url or "login" in url:
            return False
        return "intra.42.fr" in url
    except Exception:  # noqa: BLE001
        return False


def is_signin(url: str) -> bool:
    u = (url or "").lower()
    return "signin" in u or "/users/sign_in" in u


def write_catalog(path: Path, catalog: dict[str, int]) -> None:
    path.write_text(
        json.dumps(dict(sorted(catalog.items())), indent=2) + "\n",
        encoding="utf-8",
    )


def discover_slugs_from_list(page, delay: float, max_pages: int) -> list[str]:
    ordered: list[str] = []
    seen: set[str] = set()

    for page_n in range(1, max_pages + 1):
        url = PROJECTS_LIST if page_n == 1 else f"{PROJECTS_LIST}?page={page_n}"
        print(f"[lista] página {page_n}: {url}", flush=True)
        try:
            page.goto(url, wait_until="domcontentloaded", timeout=60_000)
            page.wait_for_timeout(1000)
        except Exception as exc:  # noqa: BLE001
            print(f"  erro ao abrir lista: {exc}", file=sys.stderr)
            break

        if is_signin(page.url):
            raise SystemExit(
                "Redirecionado para sign-in na lista de projetos. "
                "Faça login e volte a correr."
            )

        html = page.content()
        batch = extract_slugs(html)
        new = 0
        for slug in batch:
            key = slug.lower()
            if key in seen:
                continue
            seen.add(key)
            ordered.append(slug)
            new += 1

        print(f"  +{new} projetos (total {len(ordered)})", flush=True)
        if new == 0:
            print("  página sem projetos novos — fim da lista.", flush=True)
            break
        time.sleep(delay)

    return ordered


def scrape_projects(
    page,
    slugs: list[str],
    catalog: dict[str, int],
    *,
    lang: str,
    delay: float,
    out: Path,
    save_every: int,
) -> tuple[int, list[str]]:
    found = 0
    missing: list[str] = []
    total = len(slugs)

    for i, slug in enumerate(slugs, 1):
        url = PROJECT_URL.format(slug=slug)
        print(f"[{i}/{total}] {slug}", flush=True)
        try:
            page.goto(url, wait_until="domcontentloaded", timeout=60_000)
            page.wait_for_timeout(800)
            if is_signin(page.url):
                print("  (redirecionado para login — a parar)", file=sys.stderr)
                missing.append(slug)
                break
            html = page.content()
        except Exception as exc:  # noqa: BLE001
            print(f"  erro: {exc}", file=sys.stderr)
            missing.append(slug)
            time.sleep(delay)
            continue

        pdf_id = pick_pdf_id(html, lang)
        if pdf_id is None:
            print("  (sem PDF CDN na página)", flush=True)
            missing.append(slug)
        else:
            catalog[slug] = pdf_id
            found += 1
            print(f"  → {pdf_id}", flush=True)

        if save_every > 0 and i % save_every == 0:
            write_catalog(out, catalog)
            print(f"  (progresso gravado em {out})", flush=True)

        time.sleep(delay)

    return found, missing


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--slugs-file", type=Path, default=None)
    parser.add_argument("--out", type=Path, default=DEFAULT_OUT)
    parser.add_argument("--profile", type=Path, default=DEFAULT_PROFILE)
    parser.add_argument("--delay", type=float, default=1.5)
    parser.add_argument("--lang", default="en")
    parser.add_argument("--merge", action="store_true")
    parser.add_argument("--skip-login", action="store_true")
    parser.add_argument("--max-pages", type=int, default=50)
    parser.add_argument("--save-every", type=int, default=10)
    args = parser.parse_args()

    catalog: dict[str, int] = {}
    if args.merge and args.out.is_file():
        try:
            catalog = json.loads(args.out.read_text(encoding="utf-8"))
        except json.JSONDecodeError:
            print("aviso: --out inválido, a começar vazio", file=sys.stderr)

    args.profile.mkdir(parents=True, exist_ok=True)

    with sync_playwright() as p:
        context = p.chromium.launch_persistent_context(
            user_data_dir=str(args.profile),
            headless=False,
            viewport={"width": 1280, "height": 900},
        )
        page = context.pages[0] if context.pages else context.new_page()

        if args.skip_login:
            if not already_logged_in(page):
                print(
                    "Sessão não encontrada. Corra sem --skip-login.",
                    file=sys.stderr,
                )
                context.close()
                return 1
        else:
            try:
                page.goto(INTRA_SIGNIN, wait_until="domcontentloaded", timeout=60_000)
            except Exception as exc:  # noqa: BLE001
                print(f"aviso: não abriu sign-in ({exc})", file=sys.stderr)
            wait_for_login_manual()
            if not already_logged_in(page):
                print("Ainda parece deslogado.", file=sys.stderr)
                context.close()
                return 1

        if args.slugs_file is not None:
            if not args.slugs_file.is_file():
                print(f"slugs não encontrado: {args.slugs_file}", file=sys.stderr)
                context.close()
                return 1
            slugs = load_slugs(args.slugs_file)
        else:
            print("A descobrir projetos via /projects/list …", flush=True)
            slugs = discover_slugs_from_list(page, args.delay, args.max_pages)
            print(f"\n{len(slugs)} projetos únicos na lista.\n", flush=True)

        if not slugs:
            print("nenhum projeto encontrado", file=sys.stderr)
            context.close()
            return 1

        found, missing = scrape_projects(
            page,
            slugs,
            catalog,
            lang=args.lang,
            delay=args.delay,
            out=args.out,
            save_every=args.save_every,
        )
        context.close()

    write_catalog(args.out, catalog)
    print(f"\nEscritos {len(catalog)} ids em {args.out} ({found} nesta corrida)")
    if missing:
        print(f"Sem id ({len(missing)}):", ", ".join(missing[:30]), end="")
        print(f" … (+{len(missing) - 30})" if len(missing) > 30 else "")
    print(
        "\nSeguinte:\n"
        f"  lightyear subject import {args.out}\n"
        "  # ou: cp catalog.json <repo>/internal/subjects/catalog.json"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
