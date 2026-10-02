#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/../.."

if ! ./scripts/dev/reducer.sh setup; then
  echo "[WARN] ai-context-reducer setup failed; continuing with repository-local context tools." >&2
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "[ERROR] python3 was not found in PATH." >&2
  exit 1
fi

python3 tools/next_issue.py
python3 tools/context_tool.py remote-delta --excerpt-lines 40 > .codex/remote_delta.json
python3 tools/context_tool.py context-pack

printf '\nWork context ready:\n  .codex/next_issue.md\n  .codex/remote_delta.json\n  .codex/context_pack.md\n'
