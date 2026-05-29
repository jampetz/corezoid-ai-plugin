#!/bin/sh
# Start MCP server: use cached prebuilt binary from GitHub Releases, fall back to go run .

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)          ARCH="amd64" ;;
  arm64|aarch64)   ARCH="arm64" ;;
esac

VERSION=$(grep '"version"' "$SCRIPT_DIR/../.claude-plugin/plugin.json" 2>/dev/null \
  | sed 's/.*"version": *"\([^"]*\)".*/\1/' | head -1)

if [ -n "$VERSION" ] && { [ "$OS" = "darwin" ] || [ "$OS" = "linux" ]; } && \
   { [ "$ARCH" = "amd64" ] || [ "$ARCH" = "arm64" ]; }; then

  CACHE_BIN="$HOME/.cache/corezoid-mcp/$VERSION/convctl-${OS}-${ARCH}"

  if [ ! -x "$CACHE_BIN" ]; then
    mkdir -p "$(dirname "$CACHE_BIN")"
    URL="https://github.com/corezoid/corezoid-ai-plugin/releases/download/v${VERSION}/convctl-${OS}-${ARCH}"
    TMP="${CACHE_BIN}.tmp"
    if command -v curl >/dev/null 2>&1; then
      curl -fsSL "$URL" -o "$TMP" 2>/dev/null && mv "$TMP" "$CACHE_BIN" && chmod +x "$CACHE_BIN" || rm -f "$TMP" 2>/dev/null
    elif command -v wget >/dev/null 2>&1; then
      wget -q "$URL" -O "$TMP" 2>/dev/null && mv "$TMP" "$CACHE_BIN" && chmod +x "$CACHE_BIN" || rm -f "$TMP" 2>/dev/null
    fi
  fi

  if [ -x "$CACHE_BIN" ]; then
    exec "$CACHE_BIN" "$@"
  fi
fi

# Fallback: compile from source (requires Go)
cd "$SCRIPT_DIR" && exec go run . "$@"
