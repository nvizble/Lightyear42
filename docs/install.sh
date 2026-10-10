#!/bin/sh
# lightyear installer: one command, no sudo.
#
#   curl -fsSL https://nvizble.github.io/Lightyear42/install.sh | sh
#   curl -fsSL https://nvizble.github.io/Lightyear42/install.sh | sh -s -- --canary
#
# Options: --canary (newest pre-release), --version 1.2.1, --dir ~/bin.
# It downloads the release for this OS and CPU from GitHub, checks its
# sha256, puts the binary in ~/.local/bin and adds that folder to the PATH
# in your shell's rc file. Linux (x86_64, arm64) and macOS.
set -eu

REPO="nvizble/Lightyear42"
DIR="${LIGHTYEAR_DIR:-$HOME/.local/bin}"
CHANNEL=stable
VERSION=""

say() { printf '%s\n' "$*"; }
die() { printf 'lightyear: %s\n' "$*" >&2; exit 1; }

while [ $# -gt 0 ]; do
	case "$1" in
		--canary) CHANNEL=canary ;;
		--version) [ $# -ge 2 ] || die "--version precisa de um número (ex.: --version 1.2.1)"; VERSION="${2#v}"; shift ;;
		--dir) [ $# -ge 2 ] || die "--dir precisa de uma pasta"; DIR="$2"; shift ;;
		-h|--help) sed -n '2,11p' "$0" 2>/dev/null || true; exit 0 ;;
		*) die "opção desconhecida: $1" ;;
	esac
	shift
done

# fetch URL [FILE]: prints to stdout, or saves to FILE.
if command -v curl >/dev/null 2>&1; then
	fetch() { if [ $# -eq 2 ]; then curl -fsSL -o "$2" "$1"; else curl -fsSL "$1"; fi; }
	final_url() { curl -fsSLI -o /dev/null -w '%{url_effective}' "$1"; }
elif command -v wget >/dev/null 2>&1; then
	fetch() { if [ $# -eq 2 ]; then wget -qO "$2" "$1"; else wget -qO- "$1"; fi; }
	final_url() { wget -S --spider --max-redirect=5 "$1" 2>&1 | sed -n 's/^ *[Ll]ocation: *//p' | tail -1; }
else
	die "precisa de curl ou wget"
fi

case "$(uname -s)" in
	Linux) OS=Linux ;;
	Darwin) OS=Darwin ;;
	*) die "sistema não suportado: $(uname -s) (no Windows, baixe o .zip em https://github.com/$REPO/releases)" ;;
esac
case "$(uname -m)" in
	x86_64|amd64) ARCH=x86_64 ;;
	aarch64|arm64) ARCH=arm64 ;;
	*) die "processador não suportado: $(uname -m)" ;;
esac

# The version: the latest stable comes from the /releases/latest redirect,
# the newest canary from the releases feed (neither uses the GitHub API,
# whose limit a whole campus behind one IP runs out of).
if [ -z "$VERSION" ]; then
	if [ "$CHANNEL" = canary ]; then
		VERSION=$(fetch "https://github.com/$REPO/releases.atom" | sed -n 's|.*/releases/tag/v\([^"<]*\).*|\1|p' | head -1)
	else
		VERSION=$(final_url "https://github.com/$REPO/releases/latest" | sed -n 's|.*/releases/tag/v\([^[:space:]]*\).*|\1|p' | tail -1)
	fi
	[ -n "$VERSION" ] || die "não deu para descobrir a última versão (sem internet?)"
fi

ASSET="lightyear_${VERSION}_${OS}_${ARCH}.tar.gz"
BASE="https://github.com/$REPO/releases/download/v$VERSION"
TMP=$(mktemp -d 2>/dev/null || mktemp -d -t lightyear)
trap 'rm -rf "$TMP"' EXIT INT TERM

say "Baixando o lightyear $VERSION ($OS $ARCH)…"
fetch "$BASE/$ASSET" "$TMP/$ASSET" || die "não achei $ASSET na versão $VERSION"

if fetch "$BASE/checksums.txt" "$TMP/checksums.txt" 2>/dev/null; then
	want=$(awk -v f="$ASSET" '$2 == f {print $1}' "$TMP/checksums.txt")
	if command -v sha256sum >/dev/null 2>&1; then
		got=$(sha256sum "$TMP/$ASSET" | awk '{print $1}')
	elif command -v shasum >/dev/null 2>&1; then
		got=$(shasum -a 256 "$TMP/$ASSET" | awk '{print $1}')
	else
		got=""
	fi
	if [ -n "$want" ] && [ -n "$got" ] && [ "$want" != "$got" ]; then
		die "o download não confere com o checksums.txt (sha256 $got, esperado $want)"
	fi
fi

tar -xzf "$TMP/$ASSET" -C "$TMP" lightyear || die "o pacote veio sem o binário lightyear"
mkdir -p "$DIR"
mv -f "$TMP/lightyear" "$DIR/lightyear"
chmod 755 "$DIR/lightyear"
say "Instalado em $DIR/lightyear"

# The PATH: once, in the rc of the user's shell.
case ":$PATH:" in
	*":$DIR:"*) ;;
	*)
		# $HOME/.local/bin rather than the expanded path, when it is under home.
		show="$DIR"
		case "$DIR" in "$HOME"/*) show="\$HOME${DIR#"$HOME"}" ;; esac
		shell=$(basename "${SHELL:-sh}")
		case "$shell" in
			bash) rc="$HOME/.bashrc"; line="export PATH=\"$show:\$PATH\"" ;;
			zsh) rc="${ZDOTDIR:-$HOME}/.zshrc"; line="export PATH=\"$show:\$PATH\"" ;;
			fish) rc="$HOME/.config/fish/config.fish"; line="fish_add_path $show" ;;
			*) rc="$HOME/.profile"; line="export PATH=\"$show:\$PATH\"" ;;
		esac
		if ! grep -qsF "$show" "$rc" && ! grep -qsF "$DIR" "$rc"; then
			mkdir -p "$(dirname "$rc")"
			printf '\n# lightyear\n%s\n' "$line" >> "$rc"
			say "Pus $DIR no PATH em $rc."
		fi
		say "Abra um terminal novo (ou rode: export PATH=\"$DIR:\$PATH\") para usar o comando lightyear."
		;;
esac

"$DIR/lightyear" version 2>/dev/null | head -1 || true
say "Primeiros passos: lightyear setup && lightyear login (ou só lightyear)."
