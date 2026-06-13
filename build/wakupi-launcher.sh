#!/usr/bin/env bash
# Launcher for Wakupi: run from the per-user data directory so the app's
# relative "./data" path resolves to a stable location regardless of where
# it was launched from (menu, terminal, etc).
set -euo pipefail
DATA_HOME="${XDG_DATA_HOME:-$HOME/.local/share}/wakupi"
mkdir -p "$DATA_HOME/data"
cd "$DATA_HOME"
exec "$HOME/.local/libexec/wakupi/wakupi" "$@"
