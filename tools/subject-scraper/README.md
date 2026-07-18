# subject-scraper

Ferramenta auxiliar (Python + Playwright) para gerar o mapa `slug → pdf-id`
dos subjects da Intra (CDN).

Esta pasta vive na branch `tools/subject-scraper` e **não faz parte do
binário lightyear**. O produto Go consome o resultado em
`internal/subjects/catalog.json` (outro PR / branch).

## Setup

```bash
git checkout tools/subject-scraper
cd tools/subject-scraper
python3 -m venv .venv
source .venv/bin/activate
python3 -m pip install -r requirements.txt
python3 -m playwright install chromium
```

## Uso

```bash
# descobre projetos em https://projects.intra.42.fr/projects/list
python3 scrape_subjects.py --merge

# se a sessão Chromium já estiver guardada em .pw-profile
python3 scrape_subjects.py --merge --skip-login
```

1. Faz login na Intra no Chromium.
2. Pressiona Enter no terminal.
3. O resultado fica em `catalog.json`.

## Aplicar no lightyear

Na branch do produto (`feat/subject-catalog` / `main`):

```bash
cp tools/subject-scraper/catalog.json internal/subjects/catalog.json
# ou, com a CLI autenticada:
lightyear subject import tools/subject-scraper/catalog.json
```

## Notas

- `.venv/` e `.pw-profile/` não são commitados.
- Usa `--delay 1.5` (default) para não stressar a Intra.
- Não partilhes cookies nem o perfil do Playwright.
