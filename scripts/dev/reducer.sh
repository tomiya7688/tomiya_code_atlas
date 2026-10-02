#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/../.."

REDUCER_REPO="https://github.com/tomiya7688/ai-context-reducer.git"
REDUCER_DIR="$PWD/.dev/ai-context-reducer"
ACTION="${1:-setup}"

if ! command -v git >/dev/null 2>&1; then
  echo "[ERROR] git was not found in PATH." >&2
  exit 1
fi

mkdir -p "$PWD/.dev"

if [ -d "$REDUCER_DIR/.git" ]; then
  echo "[ai-context-reducer] updating cached development checkout..."
  if git -C "$REDUCER_DIR" fetch origin main --depth=1; then
    git -C "$REDUCER_DIR" checkout -q main
    git -C "$REDUCER_DIR" reset --hard origin/main >/dev/null
  else
    echo "[WARN] Could not fetch latest reducer; using cached checkout." >&2
  fi
else
  echo "[ai-context-reducer] cloning latest development checkout..."
  git clone --depth 1 --branch main "$REDUCER_REPO" "$REDUCER_DIR"
fi

REDUCER_SHA="$(git -C "$REDUCER_DIR" rev-parse --short HEAD)"
echo "[ai-context-reducer] revision $REDUCER_SHA"

case "$ACTION" in
  update)
    exit 0
    ;;
  setup)
    exec sh "$REDUCER_DIR/tools/setup.sh" "$PWD"
    ;;
  analyze)
    exec sh "$REDUCER_DIR/tools/analyze.sh" "$PWD"
    ;;
  *)
    echo "Usage: ./reducer.sh [setup|analyze|update]" >&2
    exit 2
    ;;
esac
