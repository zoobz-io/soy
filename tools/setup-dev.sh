#!/usr/bin/env bash
#
# setup-dev.sh — provision a dev environment for the soy repo.
#
# Installs, idempotently:
#   - build tools (gcc, make) — gcc is required for `go test -race`
#   - Go toolchain (pinned, matches go.mod toolchain directive)
#   - golangci-lint + gosec (pinned to the versions CI uses)
#
# Safe to re-run. Targets Debian/Ubuntu (apt). Root or sudo required for apt.
#
# Usage: tools/setup-dev.sh

set -euo pipefail

# --- Pins ---------------------------------------------------------------
# Go: matches the `toolchain go1.25.5` directive in go.mod. CI runs 1.24 + 1.25.
GO_VERSION="${GO_VERSION:-1.25.5}"
# golangci-lint: pinned to the version in Makefile install-tools / ci.yml.
GOLANGCI_VERSION="${GOLANGCI_VERSION:-v2.7.2}"
# gosec: Makefile uses @latest; pin here for reproducibility.
GOSEC_VERSION="${GOSEC_VERSION:-latest}"

GO_INSTALL_DIR="/usr/local"
GOBIN_DIR="$(go env GOPATH 2>/dev/null || echo "$HOME/go")/bin"

# --- Helpers ------------------------------------------------------------
log()  { printf '\033[36m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[33m!!!\033[0m %s\n' "$*" >&2; }

SUDO=""
if [ "$(id -u)" -ne 0 ]; then
    if command -v sudo >/dev/null 2>&1; then
        SUDO="sudo"
    else
        warn "not root and no sudo — apt installs may fail"
    fi
fi

# --- 1. Build tools (a full C toolchain: gcc + libc headers for the race
#        detector, make for the Makefile). build-essential pulls libc6-dev,
#        without which cgo builds fail on missing <stdlib.h>.
install_build_tools() {
    if command -v gcc >/dev/null 2>&1 && command -v make >/dev/null 2>&1 \
        && [ -f /usr/include/stdlib.h ]; then
        log "C toolchain present (gcc, make, libc headers) — skipping apt"
        return
    fi
    if ! command -v apt-get >/dev/null 2>&1; then
        warn "apt-get not found; install build-essential equivalent manually"
        return
    fi
    log "installing build-essential (gcc, make, libc6-dev) via apt"
    $SUDO apt-get update -qq
    $SUDO apt-get install -y --no-install-recommends build-essential ca-certificates >/dev/null
}

# --- 2. Go toolchain ----------------------------------------------------
arch_tag() {
    case "$(uname -m)" in
        x86_64|amd64)  echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) warn "unsupported arch $(uname -m)"; exit 1 ;;
    esac
}

install_go() {
    if command -v go >/dev/null 2>&1 && [ "$(go env GOVERSION 2>/dev/null)" = "go${GO_VERSION}" ]; then
        log "go${GO_VERSION} already installed — skipping"
        return
    fi
    local arch tarball url
    arch="$(arch_tag)"
    tarball="go${GO_VERSION}.linux-${arch}.tar.gz"
    url="https://go.dev/dl/${tarball}"
    log "installing go${GO_VERSION} (${arch}) from ${url}"
    curl -fsSL "$url" -o "/tmp/${tarball}"
    $SUDO rm -rf "${GO_INSTALL_DIR}/go"
    $SUDO tar -C "${GO_INSTALL_DIR}" -xzf "/tmp/${tarball}"
    rm -f "/tmp/${tarball}"
    export PATH="${GO_INSTALL_DIR}/go/bin:${PATH}"
}

# --- 3. Persist PATH for future shells ----------------------------------
persist_path() {
    local gobin profile
    gobin="$(go env GOPATH)/bin"
    local snippet="export PATH=\"${GO_INSTALL_DIR}/go/bin:${gobin}:\$PATH\""
    # System-wide (login shells) and ~/.bashrc (non-login/interactive).
    if [ -d /etc/profile.d ] && { [ "$(id -u)" -eq 0 ] || [ -n "$SUDO" ]; }; then
        echo "$snippet" | $SUDO tee /etc/profile.d/go.sh >/dev/null
        log "wrote /etc/profile.d/go.sh"
    fi
    if [ -f "$HOME/.bashrc" ] && ! grep -qF "${GO_INSTALL_DIR}/go/bin" "$HOME/.bashrc"; then
        echo "$snippet" >> "$HOME/.bashrc"
        log "appended PATH to ~/.bashrc"
    fi
}

# --- 4. Go-installed dev tools (lint, security) -------------------------
install_go_tools() {
    export PATH="${GO_INSTALL_DIR}/go/bin:${GOBIN_DIR}:${PATH}"
    log "installing golangci-lint ${GOLANGCI_VERSION}"
    go install "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_VERSION}"
    log "installing gosec ${GOSEC_VERSION}"
    go install "github.com/securego/gosec/v2/cmd/gosec@${GOSEC_VERSION}"
}

# --- Run ----------------------------------------------------------------
install_build_tools
install_go
persist_path
install_go_tools

# --- Report -------------------------------------------------------------
export PATH="${GO_INSTALL_DIR}/go/bin:${GOBIN_DIR}:${PATH}"
echo
log "environment ready:"
printf '  gcc            %s\n' "$(gcc --version 2>/dev/null | head -1 || echo MISSING)"
printf '  make           %s\n' "$(make --version 2>/dev/null | head -1 || echo MISSING)"
printf '  go             %s\n' "$(go version 2>/dev/null || echo MISSING)"
printf '  golangci-lint  %s\n' "$(golangci-lint --version 2>/dev/null | head -1 || echo MISSING)"
printf '  gosec          %s\n' "$(gosec --version 2>/dev/null | grep -i version | head -1 || echo MISSING)"
echo
log "note: open a new shell or 'source /etc/profile.d/go.sh' to pick up PATH"
echo
log "covered: build (race), 'make test-unit', 'make lint', 'make security', 'make test-bench', 'make check'"
warn "NOT covered: 'make test-integration' spins up DB containers via testcontainers"
warn "            and needs a running Docker daemon. Install Docker separately if you need it."
