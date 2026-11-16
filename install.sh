#!/usr/bin/env bash
#
# install.sh — build and install (or update) the `cpr` CLI on Linux/macOS.
#
# Re-run this script any time to update to the latest version: it pulls the
# newest source (when run from a git clone), rebuilds, and replaces the old
# binary in place.
#
# Prerequisites checked automatically: Go, GitHub CLI (gh), and that gh is
# authenticated.
#
# Install location defaults to /usr/local/bin. Override with:
#   INSTALL_DIR=~/.local/bin ./install.sh
#
set -euo pipefail

BIN_NAME="cpr"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

info() { printf '\033[1;36m==>\033[0m %s\n' "$1"; }
ok()   { printf '\033[1;32m ✓\033[0m %s\n' "$1"; }
warn() { printf '\033[1;33m !\033[0m %s\n' "$1"; }
err()  { printf '\033[1;31m ✗\033[0m %s\n' "$1" >&2; }

has() { command -v "$1" >/dev/null 2>&1; }

# ── 1. Prerequisites ──────────────────────────────────────────────────────────
info "Checking prerequisites"

if ! has go; then
  err "Go is not installed. Install it from https://go.dev/dl/ and try again."
  exit 1
fi
ok "Go found ($(go version | awk '{print $3}'))"

if ! has gh; then
  err "GitHub CLI (gh) is not installed. Install it from https://cli.github.com/ and try again."
  exit 1
fi
ok "GitHub CLI found ($(gh --version | head -n1 | awk '{print $3}'))"

if ! gh auth status >/dev/null 2>&1; then
  err "GitHub CLI is not authenticated. Run: gh auth login"
  exit 1
fi
ok "GitHub CLI authenticated"

# ── 2. Move into the repo (this script's directory) ─────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# ── 3. Pull the latest source for monthly updates (only if a git clone) ─────────
if [ -d .git ] && has git; then
  info "Updating source (git pull --ff-only)"
  if git pull --ff-only >/dev/null 2>&1; then
    ok "Source is up to date"
  else
    warn "Could not fast-forward (local changes or detached HEAD); building current source"
  fi
fi

# ── 4. Build ────────────────────────────────────────────────────────────────────
info "Building ${BIN_NAME}"
go mod tidy
go build -o "$BIN_NAME" .
ok "Built ./${BIN_NAME}"

# ── 5. Install (remove old, copy new) ───────────────────────────────────────────
need_sudo() {
  [ "$(id -u)" = "0" ] && return 1
  if [ -d "$INSTALL_DIR" ]; then
    [ ! -w "$INSTALL_DIR" ]
  else
    [ ! -w "$(dirname "$INSTALL_DIR")" ]
  fi
}

SUDO=""
if need_sudo; then
  SUDO="sudo"
  warn "Elevated permission required to write to ${INSTALL_DIR}; using sudo"
fi

info "Installing to ${INSTALL_DIR}/${BIN_NAME}"
$SUDO mkdir -p "$INSTALL_DIR"
$SUDO rm -f "$INSTALL_DIR/$BIN_NAME"            # remove any previous version
$SUDO install -m 0755 "$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
rm -f "$BIN_NAME"                               # clean up local build artifact
ok "Installed ${BIN_NAME} → ${INSTALL_DIR}/${BIN_NAME}"

# ── 6. PATH (make it permanent, not just this session) ────────────────────────
# Append the export to a shell profile once, so it survives new terminals.
persist_path() {
  local file="$1"
  [ -e "$file" ] || : >"$file" || return 1
  if grep -qsF "$INSTALL_DIR" "$file"; then
    return 1 # already configured
  fi
  printf '\n# Added by cpr installer\nexport PATH="%s:$PATH"\n' "$INSTALL_DIR" >>"$file"
  return 0
}

case ":$PATH:" in
  *":$INSTALL_DIR:"*)
    ok "${INSTALL_DIR} is already on your PATH"
    ;;
  *)
    # Choose the profile file for the user's login shell.
    case "$(basename "${SHELL:-sh}")" in
      zsh)  profile="$HOME/.zshrc" ;;
      bash) profile="$HOME/.bashrc" ;;
      *)    profile="$HOME/.profile" ;;
    esac
    if persist_path "$profile"; then
      ok "Added ${INSTALL_DIR} to your PATH permanently (${profile})"
      warn "Run 'source ${profile}' or open a new terminal to use 'cpr' now."
    else
      ok "${INSTALL_DIR} already configured in ${profile}"
    fi
    ;;
esac

echo
ok "Done. Run 'cpr' inside any Git repository."
