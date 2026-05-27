#!/usr/bin/env bash
# Matteo Brambilla - 2026
#
# Installs and configures the development environment for the mrtutor project.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# ── Logging ──────────────────────────────────────────────────────────────────

BOLD='\033[1m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
RESET='\033[0m'

info()    { echo -e "  ${BOLD}${GREEN}✓${RESET} $*"; }
warn()    { echo -e "  ${BOLD}${YELLOW}!${RESET} $*"; }
error()   { echo -e "  ${BOLD}${RED}✗${RESET} $*" >&2; }
section() { echo -e "\n${BOLD}$*${RESET}"; }

die() {
  error "$*"
  exit 1
}

# ── Helpers ───────────────────────────────────────────────────────────────────

require_cmd() {
  local cmd="$1"
  local install_hint="${2:-}"
  if ! command -v "$cmd" &>/dev/null; then
    error "'$cmd' not found."
    [[ -n "$install_hint" ]] && warn "Install hint: $install_hint"
    exit 1
  fi
}

# ── 1. Backend ────────────────────────────────────────────────────────────────

section "1. Backend (Go)"

require_cmd go "https://go.dev/doc/install"
info "Go $(go version | awk '{print $3}') found"

info "Tidying Go modules..."
(cd "$REPO_ROOT/api" && go mod tidy)
info "Go modules ready"

# ── 2. Database tools ─────────────────────────────────────────────────────────

section "2. Database tools"

info "Installing golang-migrate (sqlite)..."
go install -tags 'sqlite' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

require_cmd migrate "go install -tags 'sqlite' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
info "migrate $(migrate -version 2>&1) found"

# ── Done ──────────────────────────────────────────────────────────────────────

echo -e "\n${BOLD}${GREEN}Setup complete.${RESET} Happy hacking!\n"
