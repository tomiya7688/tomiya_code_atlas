"""Verify that the packaged Go CLI parses source and writes selected diagrams."""

from __future__ import annotations

import subprocess
import sys
from uuid import uuid4
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


def main() -> int:
    if len(sys.argv) != 2:
        print("使い方: go_cli_smoke.py <CLI実行ファイル>", file=sys.stderr)
        return 2

    executable = Path(sys.argv[1]).resolve()
    if not executable.is_file():
        print(f"[エラー] CLI実行ファイルが見つかりません: {executable}", file=sys.stderr)
        return 1

    source = ROOT / "tests/fixtures/backend_conformance/go.go"
    smoke_directory = ROOT / ".build" / f"go-cli-smoke-{uuid4().hex}"
    output_directory = smoke_directory / "output"
    output_directory.mkdir(parents=True, exist_ok=False)
    command = [
        str(executable),
        "analyze",
        str(source),
        "--operation",
        "call-graph",
        "--operation",
        "class-diagram",
        "--output-dir",
        str(output_directory),
    ]
    try:
        completed = subprocess.run(
            command,
            cwd=executable.parent,
            capture_output=True,
            text=True,
            encoding="utf-8",
            timeout=90,
            check=False,
        )
    except subprocess.TimeoutExpired:
        print("[エラー] Go CLIの解析が90秒以内に完了しませんでした。", file=sys.stderr)
        return 1
    if completed.returncode != 0:
        print("[エラー] Go CLIが解析に失敗しました。", file=sys.stderr)
        print(completed.stderr, file=sys.stderr)
        return 1

    expectations = {
        "go_call-graph.md": ("flowchart LR", "Worker.run"),
        "go_class-diagram.md": ("classDiagram", "Worker"),
    }
    for file_name, markers in expectations.items():
        output_path = output_directory / file_name
        if not output_path.is_file():
            print(f"[エラー] CLIの出力fileが見つかりません: {file_name}", file=sys.stderr)
            return 1
        contents = output_path.read_text(encoding="utf-8")
        if any(marker not in contents for marker in markers):
            print(f"[エラー] {file_name} に期待した図の内容がありません。", file=sys.stderr)
            return 1

    print("Go CLI smoke testに成功しました。複数operationの解析結果を出力しました。")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
